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

import "fmt"

// Accessor body rendering modes. The lowering classifies each path set
// into one of these and the plugin's template maps the mode to Go
// syntax, so the statement-level Go source (return / literal / var /
// append / range) lives in the template, not in hardcoded strings here.
const (
	// ModeSlice: a single path whose `repeated string` leaf is reached
	// through scalar hops only. The accessor returns that field's slice
	// directly: `return <SliceExpr>`.
	ModeSlice = "slice"
	// ModeLiteral: every path is scalar-only (a plain string leaf
	// reached through optional message hops). The accessor returns a
	// composite literal of the nil-safe getter chains:
	// `return []string{Exprs...}`.
	ModeLiteral = "literal"
	// ModeBuilder: at least one path traverses a container (a
	// repeated/map message hop, or a map leaf), or scalars and slices
	// mix across paths. The accessor accumulates into `out` via Stmts
	// and returns it.
	ModeBuilder = "builder"
)

// Builder statement kinds. Each maps to one line of Go the plugin's
// template renders.
const (
	// StmtAppend appends a single string expression: out = append(out, Expr).
	StmtAppend = "append"
	// StmtAppendSpread spreads a []string expression: out = append(out, Expr...).
	StmtAppendSpread = "appendSpread"
	// StmtRangeOpen opens a loop: for _, Var := range Expr {.
	StmtRangeOpen = "rangeOpen"
	// StmtClose closes the most recently opened loop: }.
	StmtClose = "close"
)

// Stmt is one statement of a ModeBuilder accessor body, kept as
// structured data so the template (not this package) owns the Go syntax.
type Stmt struct {
	Kind string // one of StmtAppend / StmtAppendSpread / StmtRangeOpen / StmtClose
	Expr string // append value, spread slice, or range source; empty for StmtClose
	Var  string // loop variable; set only for StmtRangeOpen
}

// Accessor is the template-ready description of one generated accessor
// body. Mode selects how the template renders it; only the fields
// relevant to that mode are populated:
//   - ModeSlice   -> SliceExpr
//   - ModeLiteral -> Exprs
//   - ModeBuilder -> Stmts
type Accessor struct {
	// Mode is one of ModeSlice / ModeLiteral / ModeBuilder.
	Mode string
	// SliceExpr is the getter chain returned directly in ModeSlice.
	SliceExpr string
	// Exprs are the nil-safe getter chains listed in the ModeLiteral
	// composite literal, in declaration order.
	Exprs []string
	// Stmts are the accumulator statements rendered in ModeBuilder, in
	// order, with loops already balanced (open/close).
	Stmts []Stmt
}

// Lower classifies the given non-empty set of paths into a rendering
// mode and packages the data the template needs to emit an accessor body
// on receiver expression recv (e.g. "t"). It holds no Go statement
// syntax: the actual `return`/`[]string{}`/`append`/`for` text is
// emitted by the template from the Mode and the structured fields.
//
// The mode is chosen so the generated body is as simple as the shape
// allows:
//   - ModeSlice: a single path whose `repeated string` leaf is reached
//     through scalar hops only. The getter chain already yields exactly
//     the []string wanted, so the accessor returns it directly. The
//     returned slice aliases the message's backing array, matching the
//     stock getter it delegates to.
//   - ModeLiteral: every path is scalar-only (a plain string leaf reached
//     through optional message hops). Each contributes one nil-safe
//     getter chain to a single composite literal. This is the common
//     case.
//   - ModeBuilder: any path traverses a container (repeated/map message
//     hop, or a map leaf), or scalars and slices mix across paths, so a
//     single expression cannot express the result (Go cannot spread a
//     slice or range a map inside `[]string{...}`). Falls back to an
//     accumulator with append/range statements.
func Lower(recv string, paths []*Path) *Accessor {
	a := &Accessor{}
	if len(paths) == 1 {
		if expr, ok := slicePathExpr(recv, paths[0]); ok {
			a.Mode = ModeSlice
			a.SliceExpr = expr
			return a
		}
	}
	exprs := make([]string, 0, len(paths))
	allScalar := true
	for _, p := range paths {
		expr, ok := scalarPathExpr(recv, p)
		if !ok {
			allScalar = false
			break
		}
		exprs = append(exprs, expr)
	}
	if allScalar {
		a.Mode = ModeLiteral
		a.Exprs = exprs
		return a
	}
	a.Mode = ModeBuilder
	for _, p := range paths {
		a.Stmts = append(a.Stmts, buildStmts(recv, p)...)
	}
	return a
}

// getter returns the nil-safe getter for s invoked on receiver
// expression recv, e.g. getter("t", actorStep) -> "t.GetActor()".
// Generated GetXxx() methods return the zero value on a nil receiver, so
// chaining these stays panic-free through missing intermediate hops.
func getter(recv string, s Step) string {
	return recv + ".Get" + s.GoName + "()"
}

// scalarPathExpr returns the single nil-safe getter-chain expression for
// a scalar-only path - one whose every message hop is optional (scalar)
// and whose leaf is a plain string field. It reports false when the path
// traverses any container (repeated / map) hop or leaf, in which case
// the caller must emit loop-based statements instead.
func scalarPathExpr(recv string, p *Path) (string, bool) {
	expr := recv
	for _, s := range p.Steps {
		g := getter(expr, s)
		switch s.Kind {
		case StepScalarMessage:
			expr = g
		case StepStringLeaf:
			return g, true
		default:
			return "", false
		}
	}
	return "", false
}

// slicePathExpr returns the nil-safe getter-chain expression for a path
// whose every message hop is optional (scalar) and whose leaf is a
// `repeated string` field - i.e. the getter chain already yields exactly
// the []string the accessor wants (e.g. `t.GetActors()`). When this is
// the message's only path the accessor returns that slice directly, with
// no builder. It reports false for any path that traverses a
// repeated/map message hop or whose leaf is a map (those still need a
// loop).
func slicePathExpr(recv string, p *Path) (string, bool) {
	expr := recv
	for _, s := range p.Steps {
		g := getter(expr, s)
		switch s.Kind {
		case StepScalarMessage:
			expr = g
		case StepRepeatedStringLeaf:
			return g, true
		default:
			return "", false
		}
	}
	return "", false
}

// buildStmts lowers one path into the ordered, brace-balanced statements
// a ModeBuilder accessor runs against its `out` accumulator to collect
// every annotated value reachable along p from the receiver.
//
// Consecutive scalar message hops collapse into a single nil-safe getter
// chain (a missing intermediate hop yields "" rather than panicking). A
// container hop (repeated / map) opens a `range` loop and the walk
// continues on the loop variable, so nested containers nest loops.
// Ranging over a nil slice or map is a no-op, keeping the whole body
// nil-safe.
//
// The result is structured data (no Go syntax): the template turns each
// Stmt into a line of code.
func buildStmts(recv string, p *Path) []Stmt {
	var stmts []Stmt
	expr := recv
	varN := 0
	open := 0
	for _, s := range p.Steps {
		g := getter(expr, s)
		switch s.Kind {
		case StepScalarMessage:
			expr = g
		case StepRepeatedMessage, StepMapMessage:
			varN++
			v := fmt.Sprintf("e%d", varN)
			stmts = append(stmts, Stmt{Kind: StmtRangeOpen, Var: v, Expr: g})
			expr = v
			open++
		case StepStringLeaf:
			stmts = append(stmts, Stmt{Kind: StmtAppend, Expr: g})
		case StepRepeatedStringLeaf:
			stmts = append(stmts, Stmt{Kind: StmtAppendSpread, Expr: g})
		case StepMapStringLeaf:
			varN++
			v := fmt.Sprintf("v%d", varN)
			stmts = append(stmts,
				Stmt{Kind: StmtRangeOpen, Var: v, Expr: g},
				Stmt{Kind: StmtAppend, Expr: v},
				Stmt{Kind: StmtClose},
			)
		}
	}
	for ; open > 0; open-- {
		stmts = append(stmts, Stmt{Kind: StmtClose})
	}
	return stmts
}
