package http

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/internal/testtime"
)

func TestCallHttpSuccess(t *testing.T) {
	successServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			defer req.Body.Close()

			body, err := ioutil.ReadAll(req.Body)
			if assert.NoError(t, err) {
				assert.Equal(t, []byte("world"), body)
			}

			w.Header().Set("foo", "bar")
			_, err = w.Write([]byte("great success"))
			assert.NoError(t, err)
		},
	))
	defer successServer.Close()

	httpTransport := NewTransport()
	out := httpTransport.NewSingleOutbound(successServer.URL)
	require.NoError(t, out.Start(), "failed to start outbound")
	defer out.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	req, err := http.NewRequest("GET", successServer.URL, bytes.NewReader([]byte("world")))
	//require.NoError(t, err)
	req = req.WithContext(ctx)

	client := http.Client{Transport: out}

	res, err := client.Do(req)

	require.NoError(t, err)
	defer res.Body.Close()

	foo := res.Header.Get("foo")
	assert.NotNil(t, foo, "value for foo expected")
	assert.Equal(t, "bar", foo, "foo value mismatch")

	body, err := ioutil.ReadAll(res.Body)
	if assert.NoError(t, err) {
		assert.Equal(t, []byte("great success"), body)
	}
}

func TestHTTPSuccessWithMethods(t *testing.T) {
	tests := []struct {
		desc   string
		status string
		method string
		path   string
	}{
		{
			desc:   "GET success",
			status: "success",
			method: "GET",
			path:   "/",
		},
		{
			desc:   "POST success",
			status: "lolwut",
			method: "POST",
			path:   "/",
		},
		{
			desc:   "TRACE success",
			status: "lolwut",
			method: "TRACE",
			path:   "/",
		},
		{
			desc:   "PUT success",
			status: "lolwut",
			method: "PUT",
			path:   "/",
		},
	}

	httpTransport := NewTransport()

	for _, tt := range tests {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("Status", tt.status)
				defer r.Body.Close()
			},
		))
		defer server.Close()

		out := httpTransport.NewSingleOutbound(server.URL)

		require.NoError(t, out.Start(), "failed to start outbound")
		defer out.Stop()

		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 100*testtime.Millisecond)
		defer cancel()

		req, err := http.NewRequest(tt.method, server.URL, bytes.NewReader([]byte("world")))
		require.NoError(t, err)
		req = req.WithContext(ctx)

		client := http.Client{Transport: out}

		res, err := client.Do(req)

		assert.Equal(t, res.Header.Get("Status"), tt.status, "%v: application status", tt.desc)

		if !assert.NoError(t, err, "%v: call failed", tt.desc) {
			continue
		}

		if !assert.NoError(t, res.Body.Close(), "%v: failed to close response body") {
			continue
		}
	}
}

func TestHTTPErrorWithMethods(t *testing.T) {
	tests := []struct {
		desc   string
		status string
		method string
		path   string
	}{
		{
			desc:   "GET error",
			status: "success",
			method: "GET",
			path:   "/",
		},
		{
			desc:   "POST error",
			status: "error",
			method: "POST",
			path:   "/health",
		},
		{
			desc:   "TRACE error",
			status: "lolwut",
			method: "TRACE",
			path:   "/",
		},
		{
			desc:   "PUT error",
			status: "lolwut",
			method: "PUT",
			path:   "/",
		},
	}

	httpTransport := NewTransport()

	for _, tt := range tests {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Error out", http.StatusInternalServerError)
			},
		))
		defer server.Close()

		out := httpTransport.NewSingleOutbound(server.URL)

		require.NoError(t, out.Start(), "failed to start outbound")
		defer out.Stop()

		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 100*testtime.Millisecond)
		defer cancel()

		req, err := http.NewRequest(tt.method, server.URL, bytes.NewReader([]byte("world")))
		require.NoError(t, err)
		req = req.WithContext(ctx)

		client := http.Client{Transport: out}

		res, err := client.Do(req)

		if !assert.NoError(t, err, "%v: call failed", tt.desc) {
			continue
		}

		if !assert.NoError(t, res.Body.Close(), "%v: failed to close response body") {
			continue
		}
	}
}

func TestHTTPTimeout(t *testing.T) {
	tests := []struct {
		desc    string
		method  string
		path    string
		timeout time.Duration
	}{
		{
			desc:    "GET error",
			method:  "GET",
			path:    "/",
			timeout: 0,
		},
		{
			desc:    "POST error",
			method:  "POST",
			path:    "/",
			timeout: 0,
		},
	}

	httpTransport := NewTransport()

	for _, tt := range tests {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Error out", http.StatusInternalServerError)
			},
		))
		defer server.Close()

		out := httpTransport.NewSingleOutbound(server.URL)

		require.NoError(t, out.Start(), "failed to start outbound")
		defer out.Stop()

		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, tt.timeout)
		defer cancel()

		req, err := http.NewRequest(tt.method, server.URL, bytes.NewReader([]byte("world")))
		require.NoError(t, err)
		req = req.WithContext(ctx)

		client := http.Client{Transport: out}

		res, err := client.Do(req)

		assert.Error(t, err)
		assert.Nil(t, res)
	}
}

func TestHTTPApplicationError(t *testing.T) {
	tests := []struct {
		desc     string
		status   string
		appError bool
	}{
		{
			desc:     "ok",
			status:   "success",
			appError: false,
		},
		{
			desc:     "error",
			status:   "error",
			appError: true,
		},
		{
			desc:     "not an error",
			status:   "lolwut",
			appError: false,
		},
	}

	httpTransport := NewTransport()

	for _, tt := range tests {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("Status", tt.status)
				defer r.Body.Close()
			},
		))
		defer server.Close()

		out := httpTransport.NewSingleOutbound(server.URL)

		require.NoError(t, out.Start(), "failed to start outbound")
		defer out.Stop()

		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 100*testtime.Millisecond)
		defer cancel()

		req, err := http.NewRequest("GET", server.URL, bytes.NewReader([]byte("world")))
		require.NoError(t, err)
		req = req.WithContext(ctx)

		client := http.Client{Transport: out}

		res, err := client.Do(req)

		assert.Equal(t, res.Header.Get("Status"), tt.status, "%v: application status", tt.desc)

		if !assert.NoError(t, err, "%v: call failed", tt.desc) {
			continue
		}

		if !assert.NoError(t, res.Body.Close(), "%v: failed to close response body") {
			continue
		}
	}
}

func TestHTTPCallFailures(t *testing.T) {
	notFoundServer := httptest.NewServer(http.NotFoundHandler())
	defer notFoundServer.Close()

	internalErrorServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "great sadness", http.StatusInternalServerError)
		}))
	defer internalErrorServer.Close()

	httpTransport := NewTransport()

	tests := []struct {
		url      string
		messages []string
	}{
		{"not a URL", []string{"protocol scheme"}},
		{notFoundServer.URL, []string{"404", "page not found"}},
		{internalErrorServer.URL, []string{"great sadness"}},
	}

	for _, tt := range tests {
		out := httpTransport.NewSingleOutbound(tt.url)
		require.NoError(t, out.Start(), "failed to start outbound")
		defer out.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
		defer cancel()
		req, _ := http.NewRequest("GET", internalErrorServer.URL, bytes.NewReader([]byte("world")))
		req = req.WithContext(ctx)

		client := http.Client{Transport: out}

		res, err := client.Do(req)
		assert.NoError(t, err)

		assert.NotEqual(t, res.Status, 200)
	}
}
