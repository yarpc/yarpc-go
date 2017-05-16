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

// IDLModule is a generic IDL module. For example, a thrift file or a protobuf
// one.
type IDLModule struct {
	FilePath   string      `json:"filePath"`
	SHA1       string      `json:"sha1"`
	Includes   []IDLModule `json:"includes,omitempty"`
	RawContent string      `json:"rawContent,omitempty"`
}

// Flatmap iterates recursively over the idl module and its includes. The cb
// function is called onces for every unique idl module (based on FilePath).
// Iteration is aborted if cb returns true.
func (im *IDLModule) Flatmap(cb func(*IDLModule) bool) {
	ft := newIDLFlattener(cb)
	ft.Collect(im)
}

// IDLModules is a sortable slice of IDLModule.
type IDLModules []IDLModule

func (ims IDLModules) Len() int {
	return len(ims)
}

func (ims IDLModules) Less(i int, j int) bool {
	return ims[i].FilePath < ims[j].FilePath
}

func (ims IDLModules) Swap(i int, j int) {
	ims[i], ims[j] = ims[j], ims[i]
}

// IDLTree builds an IDL tree from a slice of Module.
func (ims IDLModules) IDLTree() IDLTree {
	tb := newIDLTreeBuilder()
	for i := range ims {
		tb.Collect(&ims[i])
	}
	return tb.Root
}

// Flatmap iterates recursively over every idl modules. The cb function is
// called onces for every unique idl module (based on FilePath). Iteration is
// aborted if cb returns true.
func (ims IDLModules) Flatmap(cb func(*IDLModule) bool) {
	ft := newIDLFlattener(cb)
	for i := range ims {
		ft.Collect(&ims[i])
	}
}

// IDLTree is a tree of IDLs. A tree can have directories. Dir maps directory
// names to IDLtrees. And of course, a tree can have any number of idl modules.
type IDLTree struct {
	Dir     map[string]*IDLTree `json:"dir,omitempty"`
	Modules IDLModules          `json:"modules,omitempty"`
}

// Compact replaces any tree with a single directory and no IDLModules by it's
// single child. The the lone directory name is appended to the parent's
// directory. I would write an example, but go doc would likely fuck up
// rendering it anyway.
func (it *IDLTree) Compact() {
	for _, l1tree := range it.Dir {
		l1tree.Compact()
	}
	for l1dir, l1tree := range it.Dir {
		if len(l1tree.Dir) == 1 && len(l1tree.Modules) == 0 {
			for l2dir, l2tree := range l1tree.Dir {
				compactedDir := l1dir + "/" + l2dir
				it.Dir = map[string]*IDLTree{compactedDir: l2tree}
				break
			}
		}
	}
}

type idlFlattener struct {
	seen map[string]struct{}
	cb   func(*IDLModule) bool
}

func newIDLFlattener(cb func(*IDLModule) bool) idlFlattener {
	return idlFlattener{
		seen: make(map[string]struct{}),
		cb:   cb,
	}
}

func (idf *idlFlattener) Collect(m *IDLModule) {
	if _, ok := idf.seen[m.FilePath]; !ok {
		idf.seen[m.FilePath] = struct{}{}
		if idf.cb(m) {
			return
		}
	}
	for i := range m.Includes {
		idf.Collect(&m.Includes[i])
	}
}

type idlTreeBuilder struct {
	seen map[string]struct{}
	Root IDLTree
}

func newIDLTreeBuilder() idlTreeBuilder {
	return idlTreeBuilder{
		seen: make(map[string]struct{}),
	}
}

func (ib *idlTreeBuilder) Collect(m *IDLModule) {
	if _, ok := ib.seen[m.FilePath]; ok {
		return
	}

	ib.seen[m.FilePath] = struct{}{}
	n := &ib.Root
	parts := strings.Split(m.FilePath, "/")
	for i, part := range parts {
		if i == len(parts)-1 {
			continue
		}
		if n.Dir == nil {
			newNode := IDLTree{}
			n.Dir = map[string]*IDLTree{part: &newNode}
			n = &newNode
		} else {
			if subNode, ok := n.Dir[part]; ok {
				n = subNode
			} else {
				newNode := IDLTree{}
				n.Dir[part] = &newNode
				n = &newNode
			}
		}
	}
	n.Modules = append(n.Modules, *m)
	for i := range m.Includes {
		ib.Collect(&m.Includes[i])
	}
}
