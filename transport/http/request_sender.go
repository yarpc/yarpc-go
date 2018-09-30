package http

import "net/http"

var (
	_ sender = (*http.Client)(nil)
	_ sender = (*transportSender)(nil)
)

type sender interface {
	Do(*http.Request) (*http.Response, error)
}

type transportSender struct {
	*http.Client
}

func (t *transportSender) Do(req *http.Request) (*http.Response, error) {
	return t.Client.Transport.RoundTrip(req)
}
