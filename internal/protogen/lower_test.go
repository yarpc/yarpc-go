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
)

// step builds a Step with only the fields lowering reads: the Go name
// and the kind. Lowering never touches descriptors, so the tests need
// nothing but these literals.
func step(goName string, kind StepKind) Step {
	return Step{GoName: goName, Kind: kind}
}

func path(steps ...Step) *Path {
	return &Path{Steps: steps}
}

// TestBuildStmts asserts the structured builder statements lowered for a
// few representative paths. The statements carry no Go syntax - the
// plugin's template turns each into a line of code - so the assertions
// compare Stmt values directly. GoNames arrive already CamelCased from
// the converter.
func TestBuildStmts(t *testing.T) {
	t.Run("directField", func(t *testing.T) {
		p := path(step("Actor", StepStringLeaf))
		assert.Equal(t, []Stmt{
			{Kind: StmtAppend, Expr: "t.GetActor()"},
		}, buildStmts("t", p))
	})

	t.Run("multiStepScalarChain", func(t *testing.T) {
		p := path(
			step("Outer", StepScalarMessage),
			step("Mid", StepScalarMessage),
			step("InnerUuid", StepStringLeaf),
		)
		assert.Equal(t, []Stmt{
			{Kind: StmtAppend, Expr: "t.GetOuter().GetMid().GetInnerUuid()"},
		}, buildStmts("t", p))
	})

	t.Run("repeatedStringLeaf", func(t *testing.T) {
		p := path(step("Actors", StepRepeatedStringLeaf))
		assert.Equal(t, []Stmt{
			{Kind: StmtAppendSpread, Expr: "t.GetActors()"},
		}, buildStmts("t", p))
	})

	t.Run("mapStringLeaf", func(t *testing.T) {
		p := path(step("ActorsByRole", StepMapStringLeaf))
		assert.Equal(t, []Stmt{
			{Kind: StmtRangeOpen, Var: "v1", Expr: "t.GetActorsByRole()"},
			{Kind: StmtAppend, Expr: "v1"},
			{Kind: StmtClose},
		}, buildStmts("t", p))
	})

	t.Run("repeatedMessageHop", func(t *testing.T) {
		p := path(
			step("Items", StepRepeatedMessage),
			step("Uuid", StepStringLeaf),
		)
		assert.Equal(t, []Stmt{
			{Kind: StmtRangeOpen, Var: "e1", Expr: "t.GetItems()"},
			{Kind: StmtAppend, Expr: "e1.GetUuid()"},
			{Kind: StmtClose},
		}, buildStmts("t", p))
	})

	t.Run("mapMessageHop", func(t *testing.T) {
		p := path(
			step("ItemsById", StepMapMessage),
			step("Uuid", StepStringLeaf),
		)
		assert.Equal(t, []Stmt{
			{Kind: StmtRangeOpen, Var: "e1", Expr: "t.GetItemsById()"},
			{Kind: StmtAppend, Expr: "e1.GetUuid()"},
			{Kind: StmtClose},
		}, buildStmts("t", p))
	})

	t.Run("nestedContainerLoops", func(t *testing.T) {
		// repeated Item where Item has a repeated string leaf: a
		// container hop above a container leaf nests one loop.
		p := path(
			step("Items", StepRepeatedMessage),
			step("Actors", StepRepeatedStringLeaf),
		)
		assert.Equal(t, []Stmt{
			{Kind: StmtRangeOpen, Var: "e1", Expr: "t.GetItems()"},
			{Kind: StmtAppendSpread, Expr: "e1.GetActors()"},
			{Kind: StmtClose},
		}, buildStmts("t", p))
	})

	t.Run("repeatedMessageContainingMapStringLeaf", func(t *testing.T) {
		// repeated Item where Item has a map<string,string> leaf nests
		// the map's value loop inside the slice loop, with distinct
		// variables.
		p := path(
			step("Items", StepRepeatedMessage),
			step("ActorsByRole", StepMapStringLeaf),
		)
		assert.Equal(t, []Stmt{
			{Kind: StmtRangeOpen, Var: "e1", Expr: "t.GetItems()"},
			{Kind: StmtRangeOpen, Var: "v2", Expr: "e1.GetActorsByRole()"},
			{Kind: StmtAppend, Expr: "v2"},
			{Kind: StmtClose},
			{Kind: StmtClose},
		}, buildStmts("t", p))
	})
}

// TestLower asserts the mode classification and the data each mode
// carries: scalar-only paths collapse to a single literal, a lone
// repeated-string leaf returns its slice directly, and any container hop
// forces the builder.
func TestLower(t *testing.T) {
	scalar := func(goName string) *Path {
		return path(step(goName, StepStringLeaf))
	}

	t.Run("singleScalarIsLiteral", func(t *testing.T) {
		a := Lower("t", []*Path{scalar("Actor")})
		assert.Equal(t, ModeLiteral, a.Mode)
		assert.Equal(t, []string{"t.GetActor()"}, a.Exprs)
	})

	t.Run("multipleScalarsShareOneLiteral", func(t *testing.T) {
		a := Lower("t", []*Path{scalar("First"), scalar("Second")})
		assert.Equal(t, ModeLiteral, a.Mode)
		assert.Equal(t, []string{"t.GetFirst()", "t.GetSecond()"}, a.Exprs)
	})

	repeated := func(goName string) *Path {
		return path(step(goName, StepRepeatedStringLeaf))
	}

	t.Run("loneRepeatedStringLeafReturnsSliceDirectly", func(t *testing.T) {
		a := Lower("t", []*Path{repeated("Actors")})
		assert.Equal(t, ModeSlice, a.Mode)
		assert.Equal(t, "t.GetActors()", a.SliceExpr,
			"a sole repeated-string leaf returns the getter's slice without a builder")
	})

	t.Run("loneRepeatedStringLeafThroughScalarHopReturnsChain", func(t *testing.T) {
		p := path(
			step("Inner", StepScalarMessage),
			step("Actors", StepRepeatedStringLeaf),
		)
		a := Lower("t", []*Path{p})
		assert.Equal(t, ModeSlice, a.Mode)
		assert.Equal(t, "t.GetInner().GetActors()", a.SliceExpr)
	})

	t.Run("multipleRepeatedStringLeavesFallBackToBuilder", func(t *testing.T) {
		// The direct-return shortcut only applies to a single path; two
		// slices must be concatenated, so the builder is used.
		a := Lower("t", []*Path{repeated("Actors"), repeated("Admins")})
		assert.Equal(t, ModeBuilder, a.Mode)
		assert.Equal(t, []Stmt{
			{Kind: StmtAppendSpread, Expr: "t.GetActors()"},
			{Kind: StmtAppendSpread, Expr: "t.GetAdmins()"},
		}, a.Stmts)
	})

	t.Run("anyContainerForcesBuilderForAllPaths", func(t *testing.T) {
		// A scalar path mixed with a map leaf: the whole body must use
		// the builder style; the scalar still appends a single value.
		mapLeaf := path(step("ActorsByRole", StepMapStringLeaf))
		a := Lower("t", []*Path{scalar("Actor"), mapLeaf})
		assert.Equal(t, ModeBuilder, a.Mode)
		assert.Equal(t, []Stmt{
			{Kind: StmtAppend, Expr: "t.GetActor()"},
			{Kind: StmtRangeOpen, Var: "v1", Expr: "t.GetActorsByRole()"},
			{Kind: StmtAppend, Expr: "v1"},
			{Kind: StmtClose},
		}, a.Stmts)
	})

	t.Run("customReceiverExpression", func(t *testing.T) {
		// The receiver is a parameter, so a future consumer can lower
		// against any expression, not just "t".
		a := Lower("req", []*Path{scalar("Actor")})
		assert.Equal(t, []string{"req.GetActor()"}, a.Exprs)
	})
}
