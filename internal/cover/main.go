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

// cover is a tool that runs `go test` with cross-package coverage on this
// repository, ignoring any packages that opt out of coverage with .nocover
// files. The coverage is written to a coverage.txt file in the current
// directory.
//
// Usage
//
// Call cover with a list of one or more import paths of packages being
// tested.
//
//   cover PKG ...
//
// This must be run from the root of the project.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	log.SetFlags(0)
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

var (
	errUsage        = errors.New("usage: cover packages")
	errNoGoPackage  = errors.New("could not find a Go package in the current directory")
	errNoImportPath = fmt.Errorf("could not determine import path for the Go package in the current directory")
)

func run() error {
	packages := os.Args[1:]
	if len(packages) == 0 {
		return errUsage
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine current directory: %v", err)
	}

	rootPkg, err := build.ImportDir(cwd, 0 /* import mode */)
	if err != nil {
		return errNoGoPackage
	}

	rootImportPath := rootPkg.ImportPath
	if len(rootImportPath) == 0 {
		return errNoImportPath
	}

	// All provided packages must be under rootImport.
	rootPackagePrefix := rootImportPath + "/"
	for _, importPath := range packages {
		if importPath == rootImportPath {
			continue
		}
		if strings.HasPrefix(importPath, rootPackagePrefix) {
			continue
		}
		return fmt.Errorf("%q is not a subpackage of %q", importPath, rootImportPath)
	}

	covFile, err := ioutil.TempFile("" /* dir */, "coverage")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %v", err)
	}
	covFileName := covFile.Name()
	defer func() {
		if err := os.Remove(covFileName); err != nil {
			log.Printf("WARN: failed to remove %q: %v", covFileName, err)
		}
	}()

	if err := covFile.Close(); err != nil {
		return fmt.Errorf("failed to close %q: %v", covFileName, err)
	}

	testArgs := []string{
		"test",
		fmt.Sprintf("-coverprofile=%v", covFileName),
		"-covermode=count",
		fmt.Sprintf("-coverpkg=%v/...", rootImportPath),
	}
	testArgs = append(testArgs, packages...)
	cmd := exec.Command("go", testArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go test failed: %v", err)
	}

	outFileName := filepath.Join(cwd, "coverage.txt")
	if err := filterIgnoredPackages(cwd, rootImportPath, covFileName, outFileName); err != nil {
		return fmt.Errorf("could not filter coverage: %v", err)
	}

	return nil
}

func filterIgnoredPackages(rootDir, rootImportPath, src, dst string) (err error) {
	r, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("could not open %q for reading: %v", src, err)
	}
	defer closeFile(src, r)

	w, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("could not open %q for writing: %v", dst, err)
	}
	defer closeFile(dst, w)

	// Map from import path to whether a package is covered or not. If an
	// entry doesn't exist in this map, the status for that package isn't
	// known yet.
	shouldCover := make(map[string]bool)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		idx := strings.IndexByte(line, ':')
		if idx < 0 {
			if _, err := fmt.Fprintln(w, line); err != nil {
				return err
			}
		}

		file := line[:idx]
		if strings.Contains(file, "/internal/examples/") ||
			strings.Contains(file, "/internal/tests/") ||
			strings.Contains(file, "/mocks/") ||
			strings.Contains(file, "test/") {
			continue
		}

		importPath := filepath.Dir(file)
		cover, ok := shouldCover[importPath]
		if !ok {
			relPath, err := filepath.Rel(rootImportPath, importPath)
			if err != nil {
				return fmt.Errorf("could not make %q relative to %q: %v", importPath, rootImportPath, err)
			}

			_, err = os.Stat(filepath.Join(rootDir, relPath, ".nocover"))

			// cover a package if .nocover doesn't exist
			cover = err != nil
			shouldCover[importPath] = cover
		}

		if !cover {
			continue
		}

		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}

	return nil
}

func closeFile(n string, c io.Closer) {
	if err := c.Close(); err != nil {
		log.Printf("WARN: Failed to close %q: %v", n, err)
	}
}
