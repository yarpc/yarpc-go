package recorder

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"testing"

	"go.uber.org/yarpc/transport"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeFilename(t *testing.T) {
	assert.EqualValues(t, sanitizeFilename(`hello`), `hello`)
	assert.EqualValues(t, sanitizeFilename(`h/e\l?l%o*`), `h_e_l_l_o_`)
	assert.EqualValues(t, sanitizeFilename(`:h|e"l<l>o.`), `_h_e_l_l_o.`)
	assert.EqualValues(t, sanitizeFilename(`10€|çí¹`), `10__çí¹`)
	assert.EqualValues(t, sanitizeFilename("hel\x00lo"), `hel_lo`)
}

type randomGenerator struct {
	randsrc *rand.Rand
}

func newRandomGenerator(seed int64) randomGenerator {
	return randomGenerator{
		randsrc: rand.New(rand.NewSource(seed)),
	}
}

// Atom returns an ASCII string.
func (r *randomGenerator) Atom() string {
	length := 3 + r.randsrc.Intn(13)
	atom := make([]byte, length)
	for i := 0; i < length; i++ {
		letter := r.randsrc.Intn(2 * 26)
		if letter < 26 {
			atom[i] = 'A' + byte(letter)
		} else {
			atom[i] = 'a' + byte(letter-26)
		}
	}
	return string(atom)
}

// Headers returns a new randomized header.
func (r *randomGenerator) Headers() transport.Headers {
	headers := transport.NewHeaders()
	size := 2 + r.randsrc.Intn(6)
	for i := 0; i < size; i++ {
		headers = headers.With(r.Atom(), r.Atom())
	}
	return headers
}

// Request returns a new randomized request.
func (r *randomGenerator) Request() (transport.Request, []byte) {
	bodyData := []byte(r.Atom())

	return transport.Request{
		Caller:          r.Atom(),
		Service:         r.Atom(),
		Encoding:        transport.Encoding(r.Atom()),
		Procedure:       r.Atom(),
		Headers:         r.Headers(),
		ShardKey:        r.Atom(),
		RoutingKey:      r.Atom(),
		RoutingDelegate: r.Atom(),
		Body:            ioutil.NopCloser(bytes.NewReader(bodyData)),
	}, bodyData
}

func TestHash(t *testing.T) {
	rgen := newRandomGenerator(42)
	request, body := rgen.Request()

	recorder := NewRecorder(t)
	referenceHash := recorder.hashRequest(&request, body)

	require.Equal(t, "7195d5a712201d2a", referenceHash)

	// Caller
	r := request
	r.Caller = rgen.Atom()
	assert.NotEqual(t, recorder.hashRequest(&r, body), referenceHash)

	// Service
	r = request
	r.Service = rgen.Atom()
	assert.NotEqual(t, recorder.hashRequest(&r, body), referenceHash)

	// Encoding
	r = request
	r.Encoding = transport.Encoding(rgen.Atom())
	assert.NotEqual(t, recorder.hashRequest(&r, body), referenceHash)

	// Procedure
	r = request
	r.Procedure = rgen.Atom()
	assert.NotEqual(t, recorder.hashRequest(&r, body), referenceHash)

	// Headers
	r = request
	r.Headers = rgen.Headers()
	assert.NotEqual(t, recorder.hashRequest(&r, body), referenceHash)

	// ShardKey
	r = request
	r.ShardKey = rgen.Atom()
	assert.NotEqual(t, recorder.hashRequest(&r, body), referenceHash)

	// RoutingKey
	r = request
	r.RoutingKey = rgen.Atom()
	assert.NotEqual(t, recorder.hashRequest(&r, body), referenceHash)

	// RoutingDelegate
	r = request
	r.RoutingDelegate = rgen.Atom()
	assert.NotEqual(t, recorder.hashRequest(&r, body), referenceHash)

	// Body
	r = request
	b := []byte(rgen.Atom())
	assert.NotEqual(t, recorder.hashRequest(&r, b), referenceHash)
}
