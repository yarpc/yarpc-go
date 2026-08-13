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

// Walk walks root's fields depth-first and returns every reachable
// annotated leaf as its own path, in declaration order, along with the
// total number of paths produced. The returned slice is empty (nil) when
// no path exists.
//
// Cycle detection tracks Node identities currently on the recursion
// stack, seeded with root so a self-cycle at the root level cannot loop
// forever. Because an entry is removed once its subtree is fully walked,
// the same Node reached through two distinct sibling fields is walked
// for both (each route yields its own path), while a true cycle back to
// an ancestor still on the stack is pruned. Cycle detection spans
// container hops just as it does scalar ones.
//
// maxPaths bounds the work: chained diamond-shaped message references
// multiply the route count (k chained diamonds yield 2^k routes to the
// same leaf), so once the count passes maxPaths the walk short-circuits
// (every remaining call returns nil immediately). Callers detect
// `count > maxPaths` on the returned count and fail generation, so the
// bound caps the work done, not just the output.
func Walk(root *Node, maxPaths int) (paths []*Path, count int) {
	visited := map[*Node]bool{root: true}
	return walkNode(root, visited, &count, maxPaths), count
}

// walkNode collects the paths contributed by every field of n, in
// declaration order.
func walkNode(n *Node, visited map[*Node]bool, count *int, maxPaths int) []*Path {
	var out []*Path
	for _, f := range n.Fields {
		out = append(out, walkField(f, visited, count, maxPaths)...)
	}
	return out
}

// walkField returns the paths contributed by a single field: a leaf
// yields one single-step path; a message hop is descended into, with the
// hop prepended to every sub-path found underneath.
func walkField(f *Field, visited map[*Node]bool, count *int, maxPaths int) []*Path {
	// Path budget exhausted: stop producing paths so the walk unwinds
	// quickly instead of expanding an unbounded route set.
	if *count > maxPaths {
		return nil
	}
	step := Step{Name: f.Name, GoName: f.GoName, Kind: f.Kind}
	switch f.Kind {
	case StepStringLeaf, StepRepeatedStringLeaf, StepMapStringLeaf:
		*count++
		return []*Path{{Steps: []Step{step}}}
	case StepScalarMessage, StepRepeatedMessage, StepMapMessage:
		if f.Child == nil || visited[f.Child] {
			return nil
		}
		visited[f.Child] = true
		subs := walkNode(f.Child, visited, count, maxPaths)
		delete(visited, f.Child)
		var out []*Path
		for _, sub := range subs {
			steps := make([]Step, 0, 1+len(sub.Steps))
			steps = append(steps, step)
			steps = append(steps, sub.Steps...)
			out = append(out, &Path{Steps: steps})
		}
		return out
	}
	return nil
}
