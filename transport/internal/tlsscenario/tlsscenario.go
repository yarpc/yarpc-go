// Copyright (c) 2022 Uber Technologies, Inc.
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

package tlsscenario

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TlsScenario holds client & server tls credentials.
type TlsScenario struct {
	CAs        *x509.CertPool
	ServerCert *x509.Certificate
	ServerKey  *ecdsa.PrivateKey
	ClientCert *x509.Certificate
	ClientKey  *ecdsa.PrivateKey
}

// Create returns client and server TLS credentials generated during
// the runtime only for testing.
func Create(t *testing.T, clientValidity time.Duration, serverValidity time.Duration) TlsScenario {
	now := time.Now()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	caBytes, err := x509.CreateCertificate(
		rand.Reader,
		&x509.Certificate{
			Subject: pkix.Name{
				CommonName: "test ca",
			},
			SerialNumber:          big.NewInt(1),
			BasicConstraintsValid: true,
			IsCA:                  true,
			KeyUsage:              x509.KeyUsageCertSign,
			NotBefore:             now,
			NotAfter:              now.Add(10 * time.Minute),
		},
		&x509.Certificate{},
		caKey.Public(),
		caKey,
	)
	require.NoError(t, err)
	ca, err := x509.ParseCertificate(caBytes)
	require.NoError(t, err)

	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	serverCertBytes, err := x509.CreateCertificate(
		rand.Reader,
		&x509.Certificate{
			Subject: pkix.Name{
				CommonName: "server",
			},
			NotAfter:     now.Add(serverValidity),
			SerialNumber: big.NewInt(2),
			IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
			KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
		},
		ca,
		serverKey.Public(),
		caKey,
	)
	require.NoError(t, err)
	serverCert, err := x509.ParseCertificate(serverCertBytes)
	require.NoError(t, err)

	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	clientCertBytes, err := x509.CreateCertificate(
		rand.Reader,
		&x509.Certificate{
			Subject: pkix.Name{
				CommonName: "client",
			},
			NotAfter:     now.Add(clientValidity),
			SerialNumber: big.NewInt(3),
			KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
		},
		ca,
		clientKey.Public(),
		caKey,
	)
	require.NoError(t, err)
	clientCert, err := x509.ParseCertificate(clientCertBytes)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	pool.AddCert(ca)

	return TlsScenario{
		CAs:        pool,
		ServerCert: serverCert,
		ServerKey:  serverKey,
		ClientCert: clientCert,
		ClientKey:  clientKey,
	}
}
