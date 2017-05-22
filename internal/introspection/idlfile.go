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

package introspection

import "strings"

// IDLFile is a generic IDL file. For example, a thrift or protobuf IDL.
type IDLFile struct {
	// FilePath starts from an arbitrary root containing the file and all its
	// includes.
	FilePath string `json:"filePath"`

	// Checksum of Content. The format is: "${type}:${hex}". For example, a
	// sha1 checksum would look like:
	// "sha1:d55f7ce6138889e9e45b62df3741820777c65fcd".
	Checksum string `json:"checksum"`

	// Includes are themselves IDLFile(s).
	Includes []IDLFile `json:"includes,omitempty"`

	// Content is the IDL file content.
	Content string `json:"content,omitempty"`
}

// Flatmap iterates recursively over the idl module and its includes. The cb
// function is called onces for every unique idl module (based on FilePath).
// Iteration is aborted if cb returns true.
func (im *IDLFile) Flatmap(cb func(*IDLFile) bool) {
	ft := newIDLFlattener(cb)
	ft.Collect(im)
}

// IDLFiles is a sortable slice of IDLFile.
type IDLFiles []IDLFile

func (ims IDLFiles) Len() int {
	return len(ims)
}

func (ims IDLFiles) Less(i int, j int) bool {
	return ims[i].FilePath < ims[j].FilePath
}

func (ims IDLFiles) Swap(i int, j int) {
	ims[i], ims[j] = ims[j], ims[i]
}

// IDLTree builds an IDL tree from a slice of Module.
func (ims IDLFiles) IDLTree() IDLTree {
	tb := newIDLTreeBuilder()
	for i := range ims {
		tb.Collect(&ims[i])
	}
	return tb.Root
}

// Flatmap iterates recursively over every idl modules. The cb function is
// called onces for every unique idl module (based on FilePath). Iteration is
// aborted if cb returns true.
func (ims IDLFiles) Flatmap(cb func(*IDLFile) bool) {
	ft := newIDLFlattener(cb)
	for i := range ims {
		ft.Collect(&ims[i])
	}
}

// IDLTree is a tree of IDLs. A tree can have directories. Dir maps directory
// names to IDLtrees. And of course, a tree can have any number of idl modules.
type IDLTree struct {
	Dir   map[string]*IDLTree `json:"dir,omitempty"`
	Files IDLFiles            `json:"files,omitempty"`
}

// Compact replaces any tree with a single directory and no IDLFiles by it's
// single child. The lone directory name is appended to the parent's directory.
// Example:
//	/a/
//	- /b/
//	  - /c/
//	    - foo.thrift
// Becomes:
//	/a/b/c/:
//	- foo.thrift
// This is mainly useful for user interface rendering.
func (it *IDLTree) Compact() {
	for _, l1tree := range it.Dir {
		l1tree.Compact()
	}
	for l1dir, l1tree := range it.Dir {
		if !(len(l1tree.Dir) == 1 && len(l1tree.Files) == 0) {
			continue
		}
		for l2dir, l2tree := range l1tree.Dir {
			compactedDir := l1dir + "/" + l2dir
			it.Dir = map[string]*IDLTree{compactedDir: l2tree}
			break
		}
	}
}

type idlFlattener struct {
	seenIDLs map[string]struct{}
	cb       func(*IDLFile) bool
}

func newIDLFlattener(cb func(*IDLFile) bool) idlFlattener {
	return idlFlattener{
		seenIDLs: make(map[string]struct{}),
		cb:       cb,
	}
}

func (idf *idlFlattener) Collect(m *IDLFile) {
	if _, ok := idf.seenIDLs[m.FilePath]; !ok {
		idf.seenIDLs[m.FilePath] = struct{}{}
		if idf.cb(m) {
			return
		}
	}
	for i := range m.Includes {
		idf.Collect(&m.Includes[i])
	}
}

type idlTreeBuilder struct {
	seenIDLs map[string]struct{}
	Root     IDLTree
}

func newIDLTreeBuilder() idlTreeBuilder {
	return idlTreeBuilder{
		seenIDLs: make(map[string]struct{}),
	}
}

func (ib *idlTreeBuilder) Collect(m *IDLFile) {
	if _, ok := ib.seenIDLs[m.FilePath]; ok {
		return
	}

	ib.seenIDLs[m.FilePath] = struct{}{}
	n := &ib.Root
	segments := strings.Split(m.FilePath, "/")
	if len(segments) >= 1 {
		segments = segments[0 : len(segments)-1]
	}
	for _, segment := range segments {
		if n.Dir == nil {
			newNode := IDLTree{}
			n.Dir = map[string]*IDLTree{segment: &newNode}
			n = &newNode
		} else {
			if subNode, ok := n.Dir[segment]; ok {
				n = subNode
			} else {
				newNode := IDLTree{}
				n.Dir[segment] = &newNode
				n = &newNode
			}
		}
	}
	n.Files = append(n.Files, *m)
	for i := range m.Includes {
		ib.Collect(&m.Includes[i])
	}
}
