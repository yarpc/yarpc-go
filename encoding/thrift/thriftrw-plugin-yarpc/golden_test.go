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

package main

import (
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"go.uber.org/atomic"
	"go.uber.org/thriftrw/plugin"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This implements a test that verifies that the code in internal/tests/ is up to
// date.

const _testPackage = "go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests"

// Thrift files for which we set --sanitize-tchannel to true.
var tchannelSanitizeFor = map[string]struct{}{
	"weather.thrift": {},
}

type fakePluginServer struct {
	ln      net.Listener
	running atomic.Bool

	// Whether the next request should use --sanitize-tchannel.
	sanitizeTChannelNext atomic.Bool
}

func newFakePluginServer(t *testing.T) *fakePluginServer {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "failed to set up TCP server")

	server := &fakePluginServer{ln: ln}
	go server.serve(t)
	return server
}

func (s *fakePluginServer) Addr() string {
	return s.ln.Addr().String()
}

func (s *fakePluginServer) Stop(t *testing.T) {
	s.running.Store(false)
	if err := s.ln.Close(); err != nil {
		t.Logf("failed to stop fake plugin server: %v", err)
	}
}

func (s *fakePluginServer) SanitizeTChannel() {
	s.sanitizeTChannelNext.Store(true)
}

func (s *fakePluginServer) serve(t *testing.T) {
	s.running.Store(true)
	for s.running.Load() {
		conn, err := s.ln.Accept()
		if err != nil {
			if s.running.Load() {
				t.Logf("failed to open incoming connection: %v", err)
			}
			break
		}
		s.handle(conn)
	}
}

func (s *fakePluginServer) handle(conn net.Conn) {
	defer conn.Close()

	// The plugin expects to close both, the reader and the writer. net.Conn
	// doesn't like Close being called multiple times so we're going to no-op
	// one of the closes.
	//
	// Additionally, the plugin server writes a response for the Goodbye
	// request on exit. As in,
	//
	//  plugin.Stop():
	//    reader.Close()
	//    writer.Write(bye)
	//    writer.Close()
	//
	// We need the writer to be writeable after the reader.Close. So we'll
	// no-op the reader.Close rather than writer.Close.
	plugin.Main(&plugin.Plugin{
		Name: "yarpc",
		ServiceGenerator: g{
			SanitizeTChannel: s.sanitizeTChannelNext.Swap(false),
		},
		Reader: ioutil.NopCloser(conn),
		Writer: conn,
	})
}

func TestCodeIsUpToDate(t *testing.T) {
	// ThriftRW expects to call the thriftrw-plugin-yarpc binary. We trick it
	// into calling back into this test by setting up a fake
	// thriftrw-plugin-yarpc exectuable which uses netcat to connect back to
	// the TCP server controlled by this test. We serve the YARPC plugin on
	// that TCP connection.
	//
	// This lets us get more accurate coverage metrics for the plugin.
	fakePlugin := newFakePluginServer(t)
	defer fakePlugin.Stop(t)
	{
		tempDir, err := ioutil.TempDir("", "current-thriftrw-plugin-yarpc")
		require.NoError(t, err, "failed to create temporary directory: %v", err)
		defer os.RemoveAll(tempDir)

		oldPath := os.Getenv("PATH")
		newPath := fmt.Sprintf("%v:%v", tempDir, oldPath)
		require.NoError(t, os.Setenv("PATH", newPath),
			"failed to add %q to PATH: %v", tempDir, err)
		defer os.Setenv("PATH", oldPath)

		fakePluginPath := filepath.Join(tempDir, "thriftrw-plugin-yarpc")
		require.NoError(t,
			ioutil.WriteFile(fakePluginPath, callback(fakePlugin.Addr()), 0777),
			"failed to create thriftrw plugin script")
	}

	thriftRoot, err := filepath.Abs("internal/tests")
	require.NoError(t, err, "could not resolve absolute path to internal/tests")

	thriftFiles, err := filepath.Glob(thriftRoot + "/*.thrift")
	require.NoError(t, err)

	outputDir, err := ioutil.TempDir("", "golden-test")
	require.NoError(t, err, "failed to create temporary directory")
	defer os.RemoveAll(outputDir)

	for _, thriftFile := range thriftFiles {
		packageName := strings.TrimSuffix(filepath.Base(thriftFile), ".thrift")
		currentPackageDir := filepath.Join("internal/tests", packageName)
		newPackageDir := filepath.Join(outputDir, packageName)

		currentHash, err := dirhash(currentPackageDir)
		require.NoError(t, err, "could not hash %q", currentPackageDir)

		_, fileName := filepath.Split(thriftFile)

		// Tell the plugin whether it should --sanitize-tchannel.
		if _, ok := tchannelSanitizeFor[fileName]; ok {
			fakePlugin.SanitizeTChannel()
		}

		err = thriftrw(
			"--no-recurse",
			"--out", outputDir,
			"--pkg-prefix", _testPackage,
			"--thrift-root", thriftRoot,
			"--plugin", "yarpc",
			thriftFile,
		)
		require.NoError(t, err, "failed to generate code for %q", thriftFile)

		newHash, err := dirhash(newPackageDir)
		require.NoError(t, err, "could not hash %q", newPackageDir)

		assert.Equal(t, currentHash, newHash,
			"Generated code for %q is out of date.", thriftFile)
	}
}

// callback generates the contents of a script which connects back to the
// given TCP server.
func callback(addr string) []byte {
	i := strings.LastIndexByte(addr, ':')
	host := addr[:i]
	port, err := strconv.ParseInt(addr[i+1:], 10, 32)
	if err != nil {
		panic(err)
	}

	return []byte(fmt.Sprintf(`#!/bin/bash -e

nc %v %v
`, host, port))
}

func thriftrw(args ...string) error {
	cmd := exec.Command("thriftrw", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func dirhash(dir string) (map[string]string, error) {
	fileHashes := make(map[string]string)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		fileHash, err := hash(path)
		if err != nil {
			return fmt.Errorf("failed to hash %q: %v", path, err)
		}

		// We only care about the path relative to the directory being
		// hashed.
		path, err = filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".nocover") {
			fileHashes[path] = fileHash
		}
		return nil
	})

	return fileHashes, err
}

func hash(name string) (string, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
