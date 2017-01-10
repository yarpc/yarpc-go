// Copyright (c) 2017 Uber Technologies, Inc.
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

package main_test

import (
	"crypto/sha1"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// This implements a test that verifies that the code in testdata/ is up to
// date.

const _testPackage = "go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/testdata"

func TestMain(m *testing.M) {
	flag.Parse()

	// We put the current version of the plugin on the path first.
	outputDir, err := ioutil.TempDir("", "current-thriftrw-plugin-yarpc")
	if err != nil {
		log.Fatalf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(outputDir)

	path := os.Getenv("PATH")
	if err := os.Setenv("PATH", fmt.Sprintf("%v:%v", outputDir, path)); err != nil {
		log.Fatalf("failed to add %q to PATH: %v", outputDir, err)
	}

	cmd := exec.Command(
		"go", "build", "-o", filepath.Join(outputDir, "thriftrw-plugin-yarpc"), ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("failed to build plugin: %v", err)
	}

	os.Exit(m.Run())
}

func TestCodeIsUpToDate(t *testing.T) {
	thriftRoot, err := filepath.Abs("testdata")
	require.NoError(t, err, "could not resolve absolute path to testdata")

	thriftFiles, err := filepath.Glob(thriftRoot + "/*.thrift")
	require.NoError(t, err)

	outputDir, err := ioutil.TempDir("", "golden-test")
	require.NoError(t, err, "failed to create temporary directory")
	defer os.RemoveAll(outputDir)

	for _, thriftFile := range thriftFiles {
		packageName := strings.TrimSuffix(filepath.Base(thriftFile), ".thrift")
		currentPackageDir := filepath.Join("testdata", packageName)
		newPackageDir := filepath.Join(outputDir, packageName)

		currentHash, err := dirhash(currentPackageDir)
		require.NoError(t, err, "could not hash %q", currentPackageDir)

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

		if newHash != currentHash {
			t.Fatalf("Generated code for %q is out of date.", thriftFile)
		}
	}
}

func thriftrw(args ...string) error {
	cmd := exec.Command("thriftrw", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func dirhash(dir string) (string, error) {
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
		fileHashes[path] = fileHash
		return nil
	})
	if err != nil {
		return "", err
	}

	fileNames := make([]string, 0, len(fileHashes))
	for name := range fileHashes {
		fileNames = append(fileNames, name)
	}
	sort.Strings(fileNames)

	h := sha1.New()
	for _, name := range fileNames {
		if _, err := fmt.Fprintf(h, "%v\t%v\n", name, fileHashes[name]); err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
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
