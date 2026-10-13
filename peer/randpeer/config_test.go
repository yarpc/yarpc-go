// Copyright (c) 2026 Uber Technologies, Inc.
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

package randpeer

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/yarpcconfig"
	"go.uber.org/yarpc/yarpctest"
)

type attrs map[string]interface{}

func TestConfig(t *testing.T) {
	cfg := yarpcconfig.New()
	cfg.RegisterPeerList(Spec())
	cfg.RegisterTransport(yarpctest.FakeTransportSpec())
	config, err := cfg.LoadConfig("our-service", attrs{
		"outbounds": attrs{
			"their-service": attrs{
				"fake-transport": attrs{
					"random": attrs{
						"peers": []string{
							"1.1.1.1:1111",
							"2.2.2.2:2222",
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, config.Outbounds)
	require.NotNil(t, config.Outbounds["their-service"])
	require.NotNil(t, config.Outbounds["their-service"].Unary)
}
