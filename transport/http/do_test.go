package http

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoHttp(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte("Hello, World!\r\n"))
	}))
	defer ts.Close()

	x := NewTransport()
	o := x.NewSingleOutbound(ts.URL)

	x.Start()
	defer x.Stop()
	o.Start()
	defer o.Stop()

	client := http.Client{
		Transport: o,
	}

	req, err := http.NewRequest("GET", ts.URL, nil)
	require.NoError(t, err)
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	res, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, "404 Not Found", res.Status)
	resb, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, []byte("Hello, World!\r\n"), resb)
}