// Copyright (c) 2025 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Package recorder records & replay yarpc requests on the client side.
//
// For recording, the client must be connected and able to issue requests to a
// remote service. Every request and its response is recorded into a YAML file,
// under the directory "testdata/recordings" relative to the test directory.
//
// During replay, the client doesn't need to be connected, for any recorded
// request Recorder will return the recorded response. Any new request (ie: not
// pre-recorded) will abort the test.
//
// NewRecorder() returns a Recorder, in the mode specified by the flag
// `--recorder=replay|append|overwrite`. `replay` is the default.
//
// The new Recorder instance is a yarpc outbound middleware. It takes a
// `testing.T` or compatible as argument.
//
// Example:
//
//	func MyTest(t *testing.T) {
//	  dispatcher := yarpc.NewDispatcher(yarpc.Config{
//	  	Name: "...",
//	  	Outbounds: transport.Outbounds{
//	  		...
//	  	},
//	    OutboundMiddleware: yarpc.OutboundMiddleware {
//	  	  Unary: recorder.NewRecorder(t),
//	    },
//	  })
//	}
//
// Running the tests in append mode:
//
//	$ go test -v ./... --recorder=append
//
// The recorded messages will be stored in
// `./testdata/recordings/*.yaml`.
package recorder

import (
	"bytes"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"go.uber.org/yarpc/api/transport"
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
// abort the test by calling logger.Fatal().
type Recorder struct {
	mode       Mode
	logger     TestingT
	recordsDir string
}

const defaultRecorderDir = "testdata/recordings"
const recordComment = `# In order to update this recording, setup your external dependencies and run
# ` + "`" + `go test <insert test files here> --recorder=replay|append|overwrite` + "`\n"
const currentRecordVersion = 1

// Mode is the recording mode of the recorder.
type Mode int

const (
	// invalidMode is private and used to represent invalid modes.
	invalidMode Mode = iota

	// Replay replays stored request/response pairs, any non pre-recorded
	// requests will be rejected.
	Replay

	// Overwrite will store all request/response pairs, overwriting existing
	// records.
	Overwrite

	// Append will store all new request/response pairs and replay from
	// existing record.
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
	default:
		return fmt.Sprintf("Mode(%d)", int(m))
	}
}

// modeFromString converts an English string of a mode to a `Mode`.
func modeFromString(s string) (Mode, error) {
	switch s {
	case "replay":
		return Replay, nil
	case "overwrite":
		return Overwrite, nil
	case "append":
		return Append, nil
	}
	return invalidMode, fmt.Errorf(`invalid mode: "%s"`, s)
}

// TestingT is an interface used by the recorder for logging and reporting fatal
// errors. It is intentionally made to match with testing.T.
type TestingT interface {
	// Logf must behaves similarly to testing.T.Logf.
	Logf(format string, args ...interface{})

	// Fatal should behaves similarly to testing.T.Fatal. Namely, it must abort
	// the current test.
	Fatal(args ...interface{})
}

// NewRecorder returns a Recorder in whatever mode specified via the
// `--recorder` flag.
//
// The new Recorder instance is a yarpc unary outbound middleware. It takes a
// logger as argument compatible with `testing.T`.
//
// See package documentation for more details.
func NewRecorder(logger TestingT, opts ...Option) *Recorder {
	cwd, err := os.Getwd()
	if err != nil {
		logger.Fatal(err)
	}
	recorder := &Recorder{
		logger: logger,
	}

	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.RecordsPath != "" {
		recorder.recordsDir = cfg.RecordsPath
	} else {
		recorder.recordsDir = filepath.Join(cwd, defaultRecorderDir)
	}

	mode := cfg.Mode
	if mode == invalidMode {
		mode, err = modeFromString(*recorderFlag)
		if err != nil {
			logger.Fatal(err)
		}
	}
	recorder.SetMode(mode)
	return recorder
}

// RecordMode sets the mode.
func RecordMode(mode Mode) Option {
	return func(cfg *config) {
		cfg.Mode = mode
	}
}

// RecordsPath sets the records directory path.
func RecordsPath(path string) Option {
	return func(cfg *config) {
		cfg.RecordsPath = path
	}
}

// Option is the type used for the functional options pattern.
type Option func(*config)

type config struct {
	Mode        Mode
	RecordsPath string
}

// SetMode let you choose enable the different replay and recording modes,
// overriding the --recorder flag.
func (r *Recorder) SetMode(newMode Mode) {
	if r.mode == newMode {
		return
	}
	r.mode = newMode
	r.logger.Logf("recorder %s from/to %v", r.mode.toHumanString(), r.recordsDir)
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

func (r *Recorder) hashRequestRecord(requestRecord *requestRecord) string {
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

	ha(requestRecord.Caller)
	ha(requestRecord.Service)
	ha(string(requestRecord.Encoding))
	ha(requestRecord.Procedure)

	orderedHeadersKeys := make([]string, 0, len(requestRecord.Headers))
	for k := range requestRecord.Headers {
		orderedHeadersKeys = append(orderedHeadersKeys, k)
	}
	sort.Strings(orderedHeadersKeys)
	for _, k := range orderedHeadersKeys {
		ha(k)
		ha(requestRecord.Headers[k])
	}

	ha(requestRecord.ShardKey)
	ha(requestRecord.RoutingKey)
	ha(requestRecord.RoutingDelegate)

	_, err := hash.Write(requestRecord.Body)
	if err != nil {
		log.Fatal(err)
	}
	return fmt.Sprintf("%x", hash.Sum64())
}

func (r *Recorder) makeFilePath(request *transport.Request, hash string) string {
	s := fmt.Sprintf("%s.%s.%s.yaml", request.Service, request.Procedure, hash)
	return filepath.Join(r.recordsDir, sanitizeFilename(s))
}

// Call implements the yarpc transport outbound middleware interface
func (r *Recorder) Call(
	ctx context.Context,
	request *transport.Request,
	out transport.UnaryOutbound) (*transport.Response, error) {
	log := r.logger

	requestRecord := r.requestToRequestRecord(request)

	requestHash := r.hashRequestRecord(&requestRecord)
	filepath := r.makeFilePath(request, requestHash)

	switch r.mode {
	case Replay:
		cachedRecord, err := r.loadRecord(filepath)
		if err != nil {
			log.Fatal(err)
		}
		response := r.recordToResponse(cachedRecord)
		return &response, nil
	case Append:
		cachedRecord, err := r.loadRecord(filepath)
		if err == nil {
			response := r.recordToResponse(cachedRecord)
			return &response, nil
		}
		fallthrough
	case Overwrite:
		response, err := out.Call(ctx, request)
		if err == nil {
			cachedRecord := record{
				Version:  currentRecordVersion,
				Request:  requestRecord,
				Response: r.responseToResponseRecord(response),
			}
			r.saveRecord(filepath, &cachedRecord)
		}
		return response, err
	default:
		panic(fmt.Sprintf("invalid record mode: %v", r.mode))
	}
}

func (r *Recorder) recordToResponse(cachedRecord *record) transport.Response {
	response := transport.Response{
		Headers: transport.HeadersFromMap(cachedRecord.Response.Headers),
		Body:    io.NopCloser(bytes.NewReader(cachedRecord.Response.Body)),
	}
	return response
}

func (r *Recorder) requestToRequestRecord(request *transport.Request) requestRecord {
	requestBody, err := io.ReadAll(request.Body)
	if err != nil {
		r.logger.Fatal(err)
	}
	request.Body = io.NopCloser(bytes.NewReader(requestBody))
	return requestRecord{
		Caller:          request.Caller,
		Service:         request.Service,
		Procedure:       request.Procedure,
		Encoding:        string(request.Encoding),
		Headers:         request.Headers.Items(),
		ShardKey:        request.ShardKey,
		RoutingKey:      request.RoutingKey,
		RoutingDelegate: request.RoutingDelegate,
		Body:            requestBody,
	}
}

func (r *Recorder) responseToResponseRecord(response *transport.Response) responseRecord {
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		r.logger.Fatal(err)
	}
	response.Body = io.NopCloser(bytes.NewReader(responseBody))
	return responseRecord{
		Headers: response.Headers.Items(),
		Body:    responseBody,
	}
}

// loadRecord attempts to load a record from the given file. If the record
// cannot be found the errRecordNotFound is returned. Any other error will
// abort the current test.
func (r *Recorder) loadRecord(filepath string) (*record, error) {
	rawRecord, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, newErrRecordNotFound(err)
		}
		r.logger.Fatal(err)
	}
	var cachedRecord record
	if err := yaml.Unmarshal(rawRecord, &cachedRecord); err != nil {
		r.logger.Fatal(err)
	}

	if cachedRecord.Version != currentRecordVersion {
		r.logger.Fatal(fmt.Sprintf("unsupported record version %d (expected %d)",
			cachedRecord.Version, currentRecordVersion))
	}

	return &cachedRecord, nil
}

// saveRecord attempts to save a record to the given file, any error fails the
// current test.
func (r *Recorder) saveRecord(filepath string, cachedRecord *record) {
	if err := os.MkdirAll(defaultRecorderDir, 0775); err != nil {
		r.logger.Fatal(err)
	}

	rawRecord, err := yaml.Marshal(&cachedRecord)
	if err != nil {
		r.logger.Fatal(err)
	}

	file, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
	if err != nil {
		r.logger.Fatal(err)
	}

	if _, err := file.Write([]byte(recordComment)); err != nil {
		r.logger.Fatal(err)
	}

	if _, err := file.Write(rawRecord); err != nil {
		r.logger.Fatal(err)
	}
}

type errRecordNotFound struct {
	underlyingError error
}

func newErrRecordNotFound(underlyingError error) errRecordNotFound {
	return errRecordNotFound{underlyingError}
}

func (e errRecordNotFound) Error() string {
	return fmt.Sprintf("record not found (%s)", e.underlyingError)
}

type requestRecord struct {
	Caller          string
	Service         string
	Procedure       string
	Encoding        string
	Headers         map[string]string
	ShardKey        string
	RoutingKey      string
	RoutingDelegate string
	Body            base64blob
}

type responseRecord struct {
	Headers map[string]string
	Body    base64blob
}

type record struct {
	Version  uint
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
