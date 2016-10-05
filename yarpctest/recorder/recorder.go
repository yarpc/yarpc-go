package recorder

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"go.uber.org/yarpc/transport"
	"golang.org/x/net/context"
	"gopkg.in/yaml.v2"
)

var recorderFlag = flag.String("recorder", "replay",
	`replay: replay from recorded request/response pairs.
overwrite: record all request/response pairs, overwriting records.
append: replay existing and record new request/response pairs.`)

// Recorder records & replay yarpc requests on the client side.
//
// For recording, the client must be connected and able to issue requests to a
// remote service. Every request and its response is recorded into a YAML file,
// under the directory "testdata/recordings".
//
// During replay, the client doesn't need to be connected, for any recorded
// request Recorder will return the recorded response. Any new request will
// return an error.
type Recorder struct {
	mode       Mode
	logger     Logger
	recordsDir string
}

const recorderDir = "testdata/recordings"

// Mode the different replay and recording modes.
type Mode int

const (
	InvalidMode = iota + 1

	// Replay replays stored request/response pairs.
	Replay

	// Overwrite will record on file all request/response pairs, overwriting
	// existing records.
	Overwrite

	// Append will record on file all new request/response pairs, keeping
	// existing record without modification.
	Append
)

func (m Mode) toHumanString() string {
	switch m {
	case Replay:
		return "replaying"
	case Overwrite:
		return "recording (overwrite)"
	case Append:
		return "recording (append)"
	}
	panic("Unreachable")
}

func ModeFromString(s string) (Mode, error) {
	switch s {
	case "replay":
		return Replay, nil
	case "overwrite":
		return Overwrite, nil
	case "append":
		return Append, nil
	}
	return InvalidMode, fmt.Errorf(`invalid mode: "%s"`, s)
}

// Logger is an interface used by the recorder for logging and reporting fatal
// errors. It is intentionally made to match with testing.T.
type Logger interface {
	// Logf must behaves similarly to testing.T.Logf.
	Logf(format string, args ...interface{})

	// Fatal should behaves similarly to testing.T.Fatal. Namely, it must abort
	// the current test.
	Fatal(args ...interface{})
}

// NewRecorder returns a Recorder in whatever mode specified via the
// `recorderFlag`.
//
// The new Recorder instance is a yarpc filter middleware. It takes a logger as
// argument compatible with `testing.T`. It is designed to be used in testing
// environment:
//
// 	dispatcher, err := yarpc.NewDispatcher(yarpc.Config{name: ...},
// 		xyarpc.Filter(recorder.NewRecorder(t)))
//
// The recorded messages will be stored in
// `./testdata/recordings/*.yaml`.
func NewRecorder(logger Logger) (recorder *Recorder) {
	cwd, err := os.Getwd()
	if err != nil {
		logger.Fatal(err)
	}
	mode, err := ModeFromString(*recorderFlag)
	if err != nil {
		logger.Fatal(err)
	}
	recorder = &Recorder{
		mode:       InvalidMode,
		logger:     logger,
		recordsDir: filepath.Join(cwd, recorderDir),
	}
	recorder.SetMode(mode)
	return recorder
}

// SetMode let you choose enable the different replay and recording modes,
// overriding the --recorder flag.
func (r *Recorder) SetMode(newMode Mode) {
	if r.mode == newMode {
		return
	}
	r.mode = newMode
	r.logger.Logf("recorder %s to %v", r.mode.toHumanString(), r.recordsDir)
}

func sanitizeFilename(s string) (r string) {
	const allowedRunes = `_-.`
	return strings.Map(func(rv rune) rune {
		if unicode.IsLetter(rv) || unicode.IsNumber(rv) {
			return rv
		}
		if strings.ContainsRune(allowedRunes, rv) {
			return rv
		}
		return '_'
	}, s)
}

func (r *Recorder) hashRequest(request *transport.Request, body []byte) string {
	log := r.logger
	hash := fnv.New64a()

	ha := func(b string) {
		_, err := hash.Write([]byte(b))
		if err != nil {
			log.Fatal(err)
		}
		_, err = hash.Write([]byte("."))
		if err != nil {
			log.Fatal(err)
		}
	}

	ha(request.Caller)
	ha(request.Service)
	ha(request.Procedure)
	ha(string(request.Encoding))

	headersMap := request.Headers.Items()
	orderedHeadersKeys := make([]string, 0, len(headersMap))
	for k := range headersMap {
		orderedHeadersKeys = append(orderedHeadersKeys, k)
	}
	sort.Strings(orderedHeadersKeys)
	for _, k := range orderedHeadersKeys {
		ha(k)
		ha(headersMap[k])
	}
	_, err := hash.Write(body)
	if err != nil {
		log.Fatal(err)
	}
	return fmt.Sprintf("%x", hash.Sum64())
}

func (r *Recorder) makeFilePath(request *transport.Request, hash string) string {
	s := fmt.Sprintf("%s.%s.%s.yaml", request.Service, request.Procedure, hash)
	return filepath.Join(r.recordsDir, sanitizeFilename(s))
}

// Call implements the yarpc transport filter interface
func (r *Recorder) Call(
	ctx context.Context,
	request *transport.Request,
	out transport.Outbound,
) (*transport.Response, error) {
	log := r.logger
	requestBody, err := ioutil.ReadAll(request.Body)
	request.Body = bytes.NewReader(requestBody)
	if err != nil {
		log.Fatal(err)
	}

	requestHash := r.hashRequest(request, requestBody)
	filepath := r.makeFilePath(request, requestHash)

	switch r.mode {
	case Replay:
		return r.loadRecord(filepath)
	case Append:
		response, err := r.loadRecord(filepath)
		if !os.IsNotExist(err) {
			return response, err
		}
	}

	response, err := out.Call(ctx, request)
	if err != nil {
		return response, err
	}

	if err := r.saveRecord(request, requestBody, response, filepath); err != nil {
		log.Fatal(err)
	}
	return response, nil
}

// loadRecord attempts to load a record from the given file.
func (r *Recorder) loadRecord(filepath string) (*transport.Response, error) {
	// TODO wrap recorder errors in theirs own type.
	rawRecord, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	var cachedRecord record
	if err := yaml.Unmarshal(rawRecord, &cachedRecord); err != nil {
		return nil, err
	}
	response := transport.Response{
		Headers: transport.HeadersFromMap(cachedRecord.Response.Headers),
		Body:    ioutil.NopCloser(bytes.NewReader(cachedRecord.Response.Body)),
	}
	return &response, nil
}

// saveRecord attempts to save a record to the given file.
func (r *Recorder) saveRecord(request *transport.Request, requestBody []byte,
	response *transport.Response, filepath string) error {

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	response.Body = ioutil.NopCloser(bytes.NewReader(responseBody))

	if err := os.MkdirAll(recorderDir, 0775); err != nil {
		return err
	}

	rawRecord, err := yaml.Marshal(&record{
		Request: requestRecord{
			Caller:    request.Caller,
			Service:   request.Service,
			Procedure: request.Procedure,
			Encoding:  string(request.Encoding),
			Headers:   request.Headers.Items(),
			Body:      requestBody,
		},
		Response: responseRecord{
			Headers: response.Headers.Items(),
			Body:    responseBody,
		},
	})
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath, rawRecord, 0664)
}

type requestRecord struct {
	Caller    string
	Service   string
	Procedure string
	Encoding  string
	Headers   map[string]string
	Body      base64blob
}

type responseRecord struct {
	Headers map[string]string
	Body    base64blob
}

type record struct {
	Request  requestRecord
	Response responseRecord
}

type base64blob []byte

func (b base64blob) MarshalYAML() (interface{}, error) {
	return base64.StdEncoding.EncodeToString(b), nil
}

func (b *base64blob) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var base64encoded string
	if err := unmarshal(&base64encoded); err != nil {
		return err
	}
	decoded, err := base64.StdEncoding.DecodeString(base64encoded)
	if err != nil {
		return err
	}
	*b = decoded
	return nil
}
