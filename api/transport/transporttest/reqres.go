// Copyright (c) 2018 Uber Technologies, Inc.
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

package transporttest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"go.uber.org/yarpc/api/transport"
)

// RequestMatcher may be used in gomock argument lists to assert that two
// requests match.
//
// Requests are considered to be matching if: all their primitive parameters
// match, the headers of the received request include all the headers from the
// source request, and the contents of the request bodies are the same.
type RequestMatcher struct {
	t    *testing.T
	req  *transport.Request
	body []byte
}

// NewRequestMatcher constructs a new RequestMatcher from the given testing.T
// and request.
//
// The request's contents are read in their entirety and replaced with a
// bytes.Reader.
func NewRequestMatcher(t *testing.T, r *transport.Request) RequestMatcher {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}

	// restore a copy of the body so that the caller can still use the request
	// object
	r.Body = bytes.NewReader(body)
	return RequestMatcher{t: t, req: r, body: body}
}

// TODO: Headers like User-Agent, Content-Length, etc. make their way to the
// user-level Request object. For now, we're doing the super set check but we
// should do something more specific once yarpc/yarpc.io#2 is resolved.

// Matches checks if the given object matches the Request provided in
// NewRequestMatcher.
func (m RequestMatcher) Matches(got interface{}) bool {
	l := m.req
	r, ok := got.(*transport.Request)
	if !ok {
		panic(fmt.Sprintf("expected *transport.Request, got %v", got))
	}

	if l.ID != r.ID {
		m.t.Logf("ID mismatch: %s != %s", l.ID, r.ID)
		return false
	}

	if l.Host != r.Host {
		m.t.Logf("Host mismatch: %s != %s", l.Host, r.Host)
		return false
	}

	if l.Environment != r.Environment {
		m.t.Logf("Environment mismatch: %s != %s", l.Environment, r.Environment)
		return false
	}

	if l.Caller != r.Caller {
		m.t.Logf("Caller mismatch: %s != %s", l.Caller, r.Caller)
		return false
	}

	if l.Service != r.Service {
		m.t.Logf("Service mismatch: %s != %s", l.Service, r.Service)
		return false
	}

	if l.Transport != r.Transport {
		m.t.Logf("Transport mismatch: %s != %s", l.Transport, r.Transport)
		return false
	}

	if l.Encoding != r.Encoding {
		m.t.Logf("Encoding mismatch: %s != %s", l.Service, r.Service)
		return false
	}

	if l.Procedure != r.Procedure {
		m.t.Logf("Procedure mismatch: %s != %s", l.Procedure, r.Procedure)
		return false
	}

	if l.ShardKey != r.ShardKey {
		m.t.Logf("Shard Key mismatch: %s != %s", l.ShardKey, r.ShardKey)
		return false
	}

	if l.RoutingKey != r.RoutingKey {
		m.t.Logf("Routing Key mismatch: %s != %s", l.RoutingKey, r.RoutingKey)
		return false
	}

	if l.RoutingDelegate != r.RoutingDelegate {
		m.t.Logf("Routing Delegate mismatch: %s != %s", l.RoutingDelegate, r.RoutingDelegate)
		return false
	}

	// len check to handle nil vs empty cases gracefully.
	if l.Headers.Len() != r.Headers.Len() {
		if !reflect.DeepEqual(l.Headers, r.Headers) {
			m.t.Logf("Headers did not match:\n\t   %v\n\t!= %v", l.Headers, r.Headers)
			return false
		}
	}

	rbody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		m.t.Fatalf("failed to read body: %v", err)
	}
	r.Body = bytes.NewReader(rbody) // in case it is reused

	if !bytes.Equal(m.body, rbody) {
		m.t.Logf("Body mismatch: %v != %v", m.body, rbody)
		return false
	}

	return true
}

func (m RequestMatcher) String() string {
	return fmt.Sprintf("matches request %v with body %v", m.req, m.body)
}

// checkSuperSet checks if the items in l are all also present in r.
func checkSuperSet(l, r transport.Headers) error {
	missing := make([]string, 0, l.Len())
	for k, vl := range l.Items() {
		vr, ok := r.Get(k)
		if !ok || vr != vl {
			missing = append(missing, k)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing headers: %v", strings.Join(missing, ", "))
	}
	return nil
}

// ResponseMatcher is similar to RequestMatcher but for responses.
type ResponseMatcher struct {
	t    *testing.T
	res  *transport.Response
	body []byte
}

// NewResponseMatcher builds a new ResponseMatcher that verifies that
// responses match the given Response.
func NewResponseMatcher(t *testing.T, r *transport.Response) ResponseMatcher {
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	// restore a copy of the body so that the caller can still use the
	// response object
	r.Body = ioutil.NopCloser(bytes.NewReader(body))
	return ResponseMatcher{t: t, res: r, body: body}
}

// Matches checks if the given object matches the Response provided in
// NewResponseMatcher.
func (m ResponseMatcher) Matches(got interface{}) bool {
	l := m.res
	r, ok := got.(*transport.Response)
	if !ok {
		panic(fmt.Sprintf("expected *transport.Response, got %v", got))
	}

	if l.ID != r.ID {
		m.t.Logf("ID fields do not match: %q != %q", l.ID, r.ID)
		return false
	}
	if l.Host != r.Host {
		m.t.Logf("Host fields do not match: %q != %q", l.Host, r.Host)
		return false
	}
	if l.Environment != r.Environment {
		m.t.Logf("Environment fields do not match: %q != %q", l.Environment, r.Environment)
		return false
	}
	if l.Service != r.Service {
		m.t.Logf("Service fields do not match: %q != %q", l.Service, r.Service)
		return false
	}
	if l.ApplicationError != r.ApplicationError {
		m.t.Logf("Application errors do not match: %v != %v", l.ApplicationError, r.ApplicationError)
		return false
	}

	if err := checkSuperSet(l.Headers, r.Headers); err != nil {
		m.t.Logf("Headers mismatch: %v != %v\n\t%v", l.Headers, r.Headers, err)
		return false
	}

	rbody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		m.t.Fatalf("failed to read body: %v", err)
	}
	r.Body = ioutil.NopCloser(bytes.NewReader(rbody)) // in case it is reused

	if !bytes.Equal(m.body, rbody) {
		m.t.Logf("Body mismatch: %v != %v", m.body, rbody)
		return false
	}

	return true
}

// FakeResponseWriter is a ResponseWriter that records the headers and the body
// written to it.
type FakeResponseWriter struct {
	IsApplicationError bool
	Headers            transport.Headers
	Body               bytes.Buffer
}

// SetApplicationError for FakeResponseWriter.
func (fw *FakeResponseWriter) SetApplicationError() {
	fw.IsApplicationError = true
}

// AddHeaders for FakeResponseWriter.
func (fw *FakeResponseWriter) AddHeaders(h transport.Headers) {
	for k, v := range h.OriginalItems() {
		fw.Headers = fw.Headers.With(k, v)
	}
}

// Write for FakeResponseWriter.
func (fw *FakeResponseWriter) Write(s []byte) (int, error) {
	return fw.Body.Write(s)
}
