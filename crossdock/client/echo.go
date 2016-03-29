// Copyright (c) 2016 Uber Technologies, Inc.
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

package client

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/encoding/json"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/http"

	"golang.org/x/net/context"
)

// Echo contains a message to echo
type Echo struct {
	Token string `json:"token"`
}

// EchoBehavior asserts that a server response is the same as the request
func EchoBehavior(addr string) (string, error) {
	yarpc := yarpc.New(yarpc.Config{
		Name: "client",
		Outbounds: transport.Outbounds{
			"yarpc-test": http.NewOutbound(fmt.Sprintf("http://%v:8081", addr)),
		},
	})
	client := json.New(yarpc.Channel("yarpc-test"))
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)

	var response Echo
	token := randString(5)

	_, err := client.Call(
		&json.Request{Context: ctx, Procedure: "echo", TTL: 3 * time.Second},
		&Echo{Token: token},
		&response,
	)

	if err != nil {
		return "", fmt.Errorf("Got err: %v", err)
	}
	if response.Token != token {
		return "", fmt.Errorf("Got %v, wanted %v", response.Token, token)
	}

	return fmt.Sprintf("Server said: %v", response.Token), nil
}

func randString(length int64) string {
	bs, err := ioutil.ReadAll(io.LimitReader(rand.Reader, length))
	if err != nil {
		panic(err)
	}
	return base64.RawStdEncoding.EncodeToString(bs)
}
