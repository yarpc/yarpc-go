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

package protogen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// leaf builds a plain string leaf field.
func leaf(name, goName string) *Field {
	return &Field{Name: name, GoName: goName, Kind: StepStringLeaf}
}

// hop builds a scalar message hop into child.
func hop(name, goName string, child *Node) *Field {
	return &Field{Name: name, GoName: goName, Kind: StepScalarMessage, Child: child}
}

// names flattens the per-path step names of a walk result.
func names(paths []*Path) [][]string {
	out := make([][]string, 0, len(paths))
	for _, p := range paths {
		row := make([]string, 0, len(p.Steps))
		for _, s := range p.Steps {
			row = append(row, s.Name)
		}
		out = append(out, row)
	}
	return out
}

// TestWalk exercises the shared traversal semantics on literal graphs.
// Shape classification and annotation reading are converter concerns and
// are tested with each plugin; here every Field is already walk-relevant.
func TestWalk(t *testing.T) {
	t.Run("directLeaf", func(t *testing.T) {
		root := &Node{Fields: []*Field{leaf("actor", "Actor")}}
		paths, count := Walk(root, 100)
		require.Len(t, paths, 1)
		assert.Equal(t, 1, count)
		assert.Equal(t, [][]string{{"actor"}}, names(paths))
		assert.Equal(t, StepStringLeaf, paths[0].Steps[0].Kind)
		assert.Equal(t, "Actor", paths[0].Steps[0].GoName)
	})

	t.Run("declarationOrderPreserved", func(t *testing.T) {
		root := &Node{Fields: []*Field{
			leaf("first", "First"),
			leaf("second", "Second"),
		}}
		paths, _ := Walk(root, 100)
		assert.Equal(t, [][]string{{"first"}, {"second"}}, names(paths))
	})

	t.Run("chainOfScalarHops", func(t *testing.T) {
		inner := &Node{Fields: []*Field{leaf("uuid", "Uuid")}}
		mid := &Node{Fields: []*Field{hop("inner", "Inner", inner)}}
		root := &Node{Fields: []*Field{hop("mid", "Mid", mid)}}
		paths, _ := Walk(root, 100)
		require.Len(t, paths, 1)
		assert.Equal(t, [][]string{{"mid", "inner", "uuid"}}, names(paths))
	})

	t.Run("siblingRoutesToSharedNodeBothWalked", func(t *testing.T) {
		// The visited set is unwound after each branch, so two sibling
		// fields sharing one Node each surface their own path.
		shared := &Node{Fields: []*Field{leaf("uuid", "Uuid")}}
		root := &Node{Fields: []*Field{
			hop("primary", "Primary", shared),
			hop("secondary", "Secondary", shared),
		}}
		paths, count := Walk(root, 100)
		assert.Equal(t, 2, count)
		assert.Equal(t, [][]string{
			{"primary", "uuid"},
			{"secondary", "uuid"},
		}, names(paths))
	})

	t.Run("selfCyclePrunedSiblingLeafWins", func(t *testing.T) {
		// A node referencing itself must be pruned by the on-stack
		// visited set; its annotated sibling still surfaces.
		node := &Node{}
		node.Fields = []*Field{
			hop("loop", "Loop", node),
			leaf("id", "Id"),
		}
		paths, _ := Walk(node, 100)
		assert.Equal(t, [][]string{{"id"}}, names(paths))
	})

	t.Run("cycleBackToAncestorPruned", func(t *testing.T) {
		// root -> child -> root: the back-edge is pruned while the
		// sibling leaf under child still surfaces.
		root := &Node{}
		child := &Node{Fields: []*Field{leaf("uuid", "Uuid")}}
		child.Fields = append(child.Fields, hop("back", "Back", root))
		root.Fields = []*Field{hop("child", "Child", child)}
		paths, _ := Walk(root, 100)
		assert.Equal(t, [][]string{{"child", "uuid"}}, names(paths))
	})

	t.Run("containerHopsCarryTheirKind", func(t *testing.T) {
		item := &Node{Fields: []*Field{leaf("uuid", "Uuid")}}
		root := &Node{Fields: []*Field{
			{Name: "items", GoName: "Items", Kind: StepRepeatedMessage, Child: item},
			{Name: "by_id", GoName: "ById", Kind: StepMapMessage, Child: item},
			{Name: "tags", GoName: "Tags", Kind: StepRepeatedStringLeaf},
			{Name: "by_role", GoName: "ByRole", Kind: StepMapStringLeaf},
		}}
		paths, _ := Walk(root, 100)
		require.Len(t, paths, 4)
		assert.Equal(t, StepRepeatedMessage, paths[0].Steps[0].Kind)
		assert.Equal(t, StepMapMessage, paths[1].Steps[0].Kind)
		assert.Equal(t, StepRepeatedStringLeaf, paths[2].Steps[0].Kind)
		assert.Equal(t, StepMapStringLeaf, paths[3].Steps[0].Kind)
	})

	t.Run("nilChildSkipped", func(t *testing.T) {
		// A message hop without a resolved child contributes nothing;
		// the walk moves on to the next field.
		root := &Node{Fields: []*Field{
			hop("ghost", "Ghost", nil),
			leaf("fallback", "Fallback"),
		}}
		paths, _ := Walk(root, 100)
		assert.Equal(t, [][]string{{"fallback"}}, names(paths))
	})

	t.Run("emptyNodeYieldsNoPaths", func(t *testing.T) {
		paths, count := Walk(&Node{}, 100)
		assert.Empty(t, paths)
		assert.Zero(t, count)
	})

	t.Run("capShortCircuitsDiamondExpansion", func(t *testing.T) {
		// 7 chained diamonds expand to 2^7 = 128 routes to the leaf.
		// With maxPaths 100 the walk must report count > 100 without
		// producing all 128 paths.
		next := &Node{Fields: []*Field{leaf("uuid", "Uuid")}}
		for i := 0; i < 7; i++ {
			next = &Node{Fields: []*Field{
				hop("a", "A", next),
				hop("b", "B", next),
			}}
		}
		root := &Node{Fields: []*Field{hop("root", "Root", next)}}
		paths, count := Walk(root, 100)
		assert.Greater(t, count, 100, "the caller detects the overflow on count")
		assert.LessOrEqual(t, count, 128)
		assert.LessOrEqual(t, len(paths), count)
	})

	t.Run("underCapDiamondExpandsFully", func(t *testing.T) {
		// 6 chained diamonds expand to 2^6 = 64 routes, under the cap.
		next := &Node{Fields: []*Field{leaf("uuid", "Uuid")}}
		for i := 0; i < 6; i++ {
			next = &Node{Fields: []*Field{
				hop("a", "A", next),
				hop("b", "B", next),
			}}
		}
		root := &Node{Fields: []*Field{hop("root", "Root", next)}}
		paths, count := Walk(root, 100)
		assert.Equal(t, 64, count)
		assert.Len(t, paths, 64)
	})
}
