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

package lib

import (
	"fmt"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	gogogen "github.com/gogo/protobuf/protoc-gen-gogo/generator"

	"go.uber.org/yarpc/internal/protogen"
	"go.uber.org/yarpc/internal/protoplugin"
)

// _ActorUUIDFQN is the fully-qualified name of the FieldOptions extension
// that marks the actor UUID on a request field.
//
// The plugin recognises the option by name only. The option itself is
// defined in a separate (typically internal-monorepo) .proto file; this
// plugin does not take a build-time dependency on that definition. Whenever
// a target .proto transitively imports the file declaring this extension,
// the plugin discovers its field number from the descriptor request itself
// and emits ActorUUID() accessors for every annotated request type in the
// target file.
const _ActorUUIDFQN = "uber.security.engsec.utoken.annotations.actor_uuid"

// fieldOptionsExtendee is the Extendee value that
// FieldDescriptorProto carries for any extension of google.protobuf.FieldOptions.
const fieldOptionsExtendee = ".google.protobuf.FieldOptions"

// _maxActorUUIDPaths bounds the total number of actor_uuid paths a single
// message may expand to. Every path becomes one expression or
// statement in the generated ActorUUID() body, and chained diamond-shaped
// message references multiply the route count (k chained diamonds yield
// 2^k routes to the same leaf), so without a bound a pathological .proto
// could make the walk and the generated file exponentially large. No
// realistic schema comes anywhere near this limit; exceeding it fails
// generation with a clear error instead of emitting an enormous file.
// The walk aborts as soon as the count passes the limit, so the bound
// caps the work done, not just the output.
//
// The cap applies to every message declared in a generated file (each
// gets its own accessor when it has an annotated path), so a
// pathological message fails its file's generation even when no service
// uses it as a request type.
const _maxActorUUIDPaths = 100

// actorUUIDMethod describes a single ActorUUID() emission on a message.
// The template iterates a slice of these to emit one accessor per
// declared message that has at least one path to an annotated field.
//
// The embedded protogen.Accessor carries the rendering mode and
// structured body data (see protogen.Lower); its fields are promoted, so
// the template reads .Mode / .SliceExpr / .Exprs / .Stmts directly off
// the entry.
type actorUUIDMethod struct {
	// GoTypeName is the Go-name of the message in the package
	// being generated (e.g. "DeleteUserRequest"; "Foo_Bar" for nested).
	GoTypeName string
	*protogen.Accessor
}

// uuidFileInfo caches the per-file UUID discovery results so that
// findActorUUIDFieldNumber and newUUIDContext are each computed at most
// once per file generation, regardless of how many times the template
// invokes UUID-related helpers for the same file.
type uuidFileInfo struct {
	num int32        // 0 means the annotation is not in scope
	ctx *uuidContext // nil when num == 0
}

var (
	uuidFileInfoMu    sync.Mutex
	uuidFileInfoCache = map[*protoplugin.File]*uuidFileInfo{}
)

func getUUIDFileInfo(file *protoplugin.File) *uuidFileInfo {
	uuidFileInfoMu.Lock()
	defer uuidFileInfoMu.Unlock()
	if info, ok := uuidFileInfoCache[file]; ok {
		return info
	}
	num := findActorUUIDFieldNumber(file)
	var ctx *uuidContext
	if num != 0 {
		ctx = newUUIDContext(file)
	}
	info := &uuidFileInfo{num: num, ctx: ctx}
	uuidFileInfoCache[file] = info
	return info
}

// actorUUIDMethods returns one ActorUUID() emission per message declared
// in the target file that has at least one path to an actor_uuid-annotated
// string leaf.
//
// Emission is keyed on declaration, not on service usage. Go only allows
// a method on a type from the type's declaring package, and the declaring
// package cannot know which services use its messages as request types:
// the service may live in another file, or in another package compiled by
// a separate protoc run whose CodeGeneratorRequest never mentions this
// file. Emitting for every declared message with an annotated path is the
// only rule that guarantees a request type's accessor exists wherever the
// service lives; the extra accessors on messages never used as requests
// are harmless (every proto message has a real generated Go type, unlike
// thrift's synthesized args structs) and return coherent results.
//
// The corollary is that a package declaring annotated messages must
// itself be generated with protoc-gen-yarpc-go: if it is generated with
// the base plugin only, no accessor exists anywhere and a service package
// whose generated handler references it fails to compile.
//
// Synthetic map-entry messages are skipped: they appear in the file's
// message list but have no generated Go struct to carry a method.
//
// Returns nil (and no error) when the target file's import graph does not
// reach the option's declaration, which is the common case and means the
// plugin should not emit any accessors.
func actorUUIDMethods(info *protoplugin.TemplateInfo) ([]*actorUUIDMethod, error) {
	fi := getUUIDFileInfo(info.File)
	if fi.num == 0 {
		return nil, nil
	}
	pkgPath := info.File.GoPackage.Path
	conv := newUUIDConverter(fi.num, fi.ctx)
	var out []*actorUUIDMethod
	for _, msg := range info.File.Messages {
		if msg.GetOptions().GetMapEntry() {
			continue
		}
		node, err := conv.convert(msg)
		if err != nil {
			return nil, err
		}
		paths, count := protogen.Walk(node, _maxActorUUIDPaths)
		if count > _maxActorUUIDPaths {
			return nil, fmt.Errorf(
				"message %s expands to more than %d actor_uuid paths; "+
					"restructure the message to reduce shared sub-message fan-out",
				msg.GetName(), _maxActorUUIDPaths)
		}
		if len(paths) == 0 {
			continue
		}
		out = append(out, newActorUUIDMethod(msg.GoType(pkgPath), paths))
	}
	return out, nil
}

// serviceHasActorUUID reports whether any method of the given service has
// a request type that reaches at least one actor_uuid-annotated leaf. The
// server template uses it to gate the service-wide handler struct's
// validator field and the validator extraction in the Build/Fx entry
// points.
func serviceHasActorUUID(info *protoplugin.TemplateInfo, service *protoplugin.Service) bool {
	fi := getUUIDFileInfo(info.File)
	if fi.num == 0 {
		return false
	}
	for _, m := range service.Methods {
		if methodHasActorUUIDNum(m, fi.num, fi.ctx) {
			return true
		}
	}
	return false
}

// methodHasActorUUID reports whether the given method's request type
// reaches at least one actor_uuid-annotated leaf. The server template
// uses it to gate the per-method validator call (which invokes
// request.ActorUUID()), so it must stay in lockstep with the accessor
// emission in actorUUIDMethods.
func methodHasActorUUID(info *protoplugin.TemplateInfo, method *protoplugin.Method) bool {
	fi := getUUIDFileInfo(info.File)
	if fi.num == 0 {
		return false
	}
	return methodHasActorUUIDNum(method, fi.num, fi.ctx)
}

// methodHasActorUUIDNum is the shared core of serviceHasActorUUID and
// methodHasActorUUID: it reports whether the method's request type has at
// least one path to an annotated leaf, reusing an already-resolved field
// number and context. A conversion failure reports false here; the same
// failure surfaces as a hard error through actorUUIDMethods, which fails
// the file's generation.
func methodHasActorUUIDNum(method *protoplugin.Method, num int32, ctx *uuidContext) bool {
	if method == nil {
		return false
	}
	req := method.RequestType
	if req == nil {
		return false
	}
	node, err := newUUIDConverter(num, ctx).convert(req)
	if err != nil {
		return false
	}
	paths, _ := protogen.Walk(node, _maxActorUUIDPaths)
	return len(paths) > 0
}

// newActorUUIDMethod lowers the given non-empty set of paths into the
// template-ready accessor description for the message named goType. The
// mode classification and statement construction live in
// protogen.Lower; the receiver expression is "t", matching the
// `func (t *X) ActorUUID()` signature the template emits.
func newActorUUIDMethod(goType string, paths []*protogen.Path) *actorUUIDMethod {
	return &actorUUIDMethod{GoTypeName: goType, Accessor: protogen.Lower("t", paths)}
}

// uuidConverter builds the descriptor-free protogen.Node graph that
// protogen.Walk traverses, from protoplugin messages. It owns every
// gogo-specific concern of the walk: classifying field shapes from
// label/type enums, resolving type references and synthetic map entries
// through the uuidContext, reading the actor_uuid annotation via gogo's
// extension machinery, and computing Go getter names with gogo's own
// CamelCase (so the emitted accessor always matches the struct fields
// gogoslick generates).
//
// The memo shares one *protogen.Node per source message, which both
// makes conversion of cyclic or diamond-shaped schemas terminate (a
// message is converted once, before its fields are filled in) and gives
// protogen.Walk the Node identities its cycle detection keys on.
//
// The produced graph holds only walk-relevant fields, in declaration
// order: message-typed hops and actor_uuid-annotated string-valued
// leaves. A leaf is a plain `string`, a `repeated string`, or a
// `map<K, string>` field carrying the annotation. Annotations on any
// other shape (non-string scalars, repeated non-string scalars, or a
// message-typed field itself) are silently dropped, though message-typed
// fields are still descended into looking for string leaves inside.
// Message-typed fields whose type reference does not resolve are
// likewise dropped.
//
// An annotation-read failure (see hasActorUUID) aborts the conversion:
// the error propagates up and the caller fails generation with it.
type uuidConverter struct {
	num  int32
	ctx  *uuidContext
	memo map[*protoplugin.Message]*protogen.Node
}

func newUUIDConverter(num int32, ctx *uuidContext) *uuidConverter {
	return &uuidConverter{
		num:  num,
		ctx:  ctx,
		memo: map[*protoplugin.Message]*protogen.Node{},
	}
}

// convert returns the memoized Node for msg, building it (and,
// transitively, every message reachable from it) on first access.
func (c *uuidConverter) convert(msg *protoplugin.Message) (*protogen.Node, error) {
	if n, ok := c.memo[msg]; ok {
		return n, nil
	}
	n := &protogen.Node{}
	// Memoize before filling in the fields so a cyclic reference back to
	// msg resolves to n instead of recursing forever.
	c.memo[msg] = n
	for _, f := range msg.Fields {
		nf, err := c.convertField(f)
		if err != nil {
			return nil, err
		}
		if nf != nil {
			n.Fields = append(n.Fields, nf)
		}
	}
	return n, nil
}

// convertField returns the neutral field for f, or nil when f is not
// walk-relevant (see the uuidConverter doc for what is dropped).
func (c *uuidConverter) convertField(f *protoplugin.Field) (*protogen.Field, error) {
	isRepeated := f.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REPEATED

	if f.GetType() != descriptor.FieldDescriptorProto_TYPE_MESSAGE {
		// Scalar field. Only a string (or repeated string) carrying the
		// annotation is a valid leaf; any other scalar is dropped.
		if f.GetType() != descriptor.FieldDescriptorProto_TYPE_STRING {
			return nil, nil
		}
		annotated, err := hasActorUUID(f.GetOptions(), c.num)
		if err != nil {
			return nil, err
		}
		if !annotated {
			return nil, nil
		}
		kind := protogen.StepStringLeaf
		if isRepeated {
			kind = protogen.StepRepeatedStringLeaf
		}
		return c.newField(f, kind, nil), nil
	}

	inner := c.ctx.lookupMessage(f.GetTypeName())
	if inner == nil {
		return nil, nil
	}

	// A map<K, V> is encoded as a repeated synthetic entry message
	// carrying map_entry = true with fields key (1) and value (2).
	if inner.GetOptions().GetMapEntry() {
		return c.convertMapField(f, inner)
	}

	// Plain message hop, scalar or repeated.
	kind := protogen.StepScalarMessage
	if isRepeated {
		kind = protogen.StepRepeatedMessage
	}
	child, err := c.convert(inner)
	if err != nil {
		return nil, err
	}
	return c.newField(f, kind, child), nil
}

// convertMapField returns the neutral field for a map field f whose
// synthetic entry message is `entry`. A map<K, string> surfaces its
// values as a single leaf when annotated; a map<K, Msg> is a hop into
// the value message.
func (c *uuidConverter) convertMapField(f *protoplugin.Field, entry *protoplugin.Message) (*protogen.Field, error) {
	val := mapValueField(entry)
	if val == nil {
		return nil, nil
	}
	switch val.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_STRING:
		// map<K, string>: the values are the leaves. The annotation
		// lives on the map field itself, not the synthetic entry.
		annotated, err := hasActorUUID(f.GetOptions(), c.num)
		if err != nil {
			return nil, err
		}
		if !annotated {
			return nil, nil
		}
		return c.newField(f, protogen.StepMapStringLeaf, nil), nil
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		// map<K, Msg>: hop into the value message.
		valMsg := c.ctx.lookupMessage(val.GetTypeName())
		if valMsg == nil {
			return nil, nil
		}
		child, err := c.convert(valMsg)
		if err != nil {
			return nil, err
		}
		return c.newField(f, protogen.StepMapMessage, child), nil
	}
	return nil, nil
}

// newField packages f into a neutral field of the given kind. The GoName
// uses gogo's CamelCase - the same normalisation gogoslick applies when
// naming the generated struct field - so the getter the accessor emits
// ("Get" + GoName + "()") always exists on the generated type.
func (c *uuidConverter) newField(f *protoplugin.Field, kind protogen.StepKind, child *protogen.Node) *protogen.Field {
	return &protogen.Field{
		Name:   f.GetName(),
		GoName: gogogen.CamelCase(f.GetName()),
		Kind:   kind,
		Child:  child,
	}
}

// mapValueField returns the `value` field (field number 2) of a synthetic
// proto map entry message, or nil if it cannot be found. The walker uses
// the value field's type to decide whether a map is a string leaf or a
// message hop.
func mapValueField(entry *protoplugin.Message) *protoplugin.Field {
	for _, f := range entry.Fields {
		if f.GetNumber() == 2 || f.GetName() == "value" {
			return f
		}
	}
	return nil
}

// uuidContext is a per-generation resolver from a message's
// fully-qualified proto name (e.g. ".pkg.Outer.Inner") to its
// *protoplugin.Message. The plugin's registry already has this map but
// does not expose it, so the UUID walker rebuilds an equivalent index
// once at the start of every generation request and reuses it for every
// recursion step.
type uuidContext struct {
	msgByFQMN map[string]*protoplugin.Message
}

func newUUIDContext(file *protoplugin.File) *uuidContext {
	c := &uuidContext{msgByFQMN: map[string]*protoplugin.Message{}}
	c.indexFile(file)
	for _, dep := range file.TransitiveDependencies {
		c.indexFile(dep)
	}
	return c
}

// indexFile registers every (top-level and nested) message in f under its
// fully-qualified name. The protoplugin registry already flattens nested
// types into f.Messages, so a single pass is enough.
func (c *uuidContext) indexFile(f *protoplugin.File) {
	for _, m := range f.Messages {
		c.msgByFQMN[m.FQMN()] = m
	}
}

func (c *uuidContext) lookupMessage(typeName string) *protoplugin.Message {
	if typeName == "" {
		return nil
	}
	return c.msgByFQMN[typeName]
}

// findActorUUIDFieldNumber walks the target file and its transitive
// dependencies for an extension of google.protobuf.FieldOptions whose
// fully-qualified name is _ActorUUIDFQN, and returns its field number.
//
// Returns 0 if the option is not in scope (the .proto being generated did
// not transitively import the file declaring the option), in which case
// there is nothing to do.
func findActorUUIDFieldNumber(f *protoplugin.File) int32 {
	if n := findActorUUIDFieldNumberInFile(f); n != 0 {
		return n
	}
	for _, dep := range f.TransitiveDependencies {
		if n := findActorUUIDFieldNumberInFile(dep); n != 0 {
			return n
		}
	}
	return 0
}

func findActorUUIDFieldNumberInFile(f *protoplugin.File) int32 {
	pkg := f.GetPackage()
	if n := findActorUUIDInExtensions(pkg, f.GetExtension()); n != 0 {
		return n
	}
	for _, msg := range f.GetMessageType() {
		if n := findActorUUIDInNestedExtensions(pkg, msg); n != 0 {
			return n
		}
	}
	return 0
}

// findActorUUIDInExtensions scans a flat slice of extension declarations
// whose containing scope is `scope` (the proto package or a containing
// message FQN) for one that extends FieldOptions and matches _ActorUUIDFQN.
func findActorUUIDInExtensions(scope string, exts []*descriptor.FieldDescriptorProto) int32 {
	for _, ext := range exts {
		if ext.GetExtendee() != fieldOptionsExtendee {
			continue
		}
		fqn := ext.GetName()
		if scope != "" {
			fqn = scope + "." + fqn
		}
		if fqn == _ActorUUIDFQN {
			return ext.GetNumber()
		}
	}
	return 0
}

// findActorUUIDInNestedExtensions recurses into a message's nested
// extension declarations.
func findActorUUIDInNestedExtensions(parentScope string, msg *descriptor.DescriptorProto) int32 {
	scope := parentScope
	if name := msg.GetName(); name != "" {
		if scope == "" {
			scope = name
		} else {
			scope = scope + "." + name
		}
	}
	if n := findActorUUIDInExtensions(scope, msg.GetExtension()); n != 0 {
		return n
	}
	for _, nested := range msg.GetNestedType() {
		if n := findActorUUIDInNestedExtensions(scope, nested); n != 0 {
			return n
		}
	}
	return 0
}

// hasActorUUID reports whether opts carries the FieldOptions extension
// with the given field number set to true.
//
// A non-nil error means the annotation's presence could not be
// determined - the descriptor at this field number belongs to an
// unrelated extension, or the option bytes do not decode as the expected
// bool. Callers must fail generation on it: treating an annotated field
// as unannotated would silently drop the security annotation, which is
// exactly the fail-open this error exists to prevent. Only a genuinely
// absent extension reports (false, nil).
func hasActorUUID(opts *descriptor.FieldOptions, num int32) (bool, error) {
	if opts == nil || num == 0 {
		return false, nil
	}
	desc, err := actorUUIDExtensionDesc(num)
	if err != nil {
		return false, err
	}
	v, err := proto.GetExtension(opts, desc)
	if err == proto.ErrMissingExtension {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf(
			"reading actor_uuid annotation (FieldOptions extension %d): %v", num, err)
	}
	b, ok := v.(*bool)
	if !ok {
		return false, fmt.Errorf(
			"actor_uuid annotation (FieldOptions extension %d) decoded to %T, expected *bool", num, v)
	}
	return b != nil && *b, nil
}

// actorUUIDExtensionCache memoises the *ExtensionDesc used to read the
// actor_uuid extension at a given field number. Only ever holds a single
// entry because per-process there is only ever one actor_uuid extension.
var (
	actorUUIDExtensionMu    sync.Mutex
	actorUUIDExtensionCache = map[int32]*proto.ExtensionDesc{}
)

// actorUUIDExtensionDesc returns the *ExtensionDesc used to read the
// actor_uuid extension at the given field number.
//
// The plugin must operate without a build-time dependency on the Go
// package generated for the extension. To do that:
//
//   - If gogo proto already has an ExtensionDesc registered at this field
//     number on FieldOptions (because some linked package happens to
//     have registered it - e.g. a test binary, or a downstream binary
//     that does import the option's .pb.go), reuse it - but only after
//     verifying it actually describes this extension (same
//     fully-qualified name, bool-typed). gogo rejects two non-identical
//     descriptors at the same field number ("proto: descriptor
//     conflict"), so an unrelated extension parked at the number cannot
//     be worked around: it fails generation loudly instead of silently
//     misreading (or dropping) the annotation.
//   - Otherwise build an ExtensionDesc on the fly that matches what the
//     option's .pb.go would have generated (varint-typed bool at the
//     given field number) and cache it for the lifetime of the process.
//     Reusing the same *ExtensionDesc on every call is required: gogo
//     proto caches the descriptor pointer on first GetExtension and
//     refuses to accept a different pointer at the same field number on
//     later calls.
func actorUUIDExtensionDesc(num int32) (*proto.ExtensionDesc, error) {
	actorUUIDExtensionMu.Lock()
	defer actorUUIDExtensionMu.Unlock()
	if desc, ok := actorUUIDExtensionCache[num]; ok {
		return desc, nil
	}
	if desc := proto.RegisteredExtensions((*descriptor.FieldOptions)(nil))[num]; desc != nil {
		if err := validateRegisteredExtensionDesc(desc, num); err != nil {
			return nil, err
		}
		actorUUIDExtensionCache[num] = desc
		return desc, nil
	}
	desc := &proto.ExtensionDesc{
		ExtendedType:  (*descriptor.FieldOptions)(nil),
		ExtensionType: (*bool)(nil),
		Field:         num,
		Name:          _ActorUUIDFQN,
		Tag:           fmt.Sprintf("varint,%d,opt,name=actor_uuid", num),
	}
	actorUUIDExtensionCache[num] = desc
	return desc, nil
}

// validateRegisteredExtensionDesc checks that a descriptor found in gogo's
// global extension registry at actor_uuid's field number really describes
// the actor_uuid extension (same fully-qualified name, bool-typed).
//
// The check is needed because the registry is keyed by bare field number
// and holds every FieldOptions extension registered by packages linked
// into this binary (notably gogoproto's, at 65001-65013), so the entry at
// actor_uuid's number is not necessarily actor_uuid. If the option's
// .proto declared actor_uuid at a number one of those unrelated
// extensions also uses, reading the option through the registered
// descriptor would misdecode it under the wrong name and type. Nor can
// the plugin fall back to its own descriptor: gogo refuses to read the
// same field number through a second, different descriptor pointer
// ("proto: descriptor conflict"). With no safe way to read the
// annotation, the collision must fail generation.
func validateRegisteredExtensionDesc(desc *proto.ExtensionDesc, num int32) error {
	if desc.Name != _ActorUUIDFQN {
		return fmt.Errorf(
			"actor_uuid extension field number collision: the descriptor set assigns "+
				"field number %d to %s, but this plugin binary has the unrelated extension "+
				"%s registered at that number on google.protobuf.FieldOptions; "+
				"declare actor_uuid at a field number no linked extension uses",
			num, _ActorUUIDFQN, desc.Name)
	}
	if _, ok := desc.ExtensionType.(*bool); !ok {
		return fmt.Errorf(
			"actor_uuid extension registered at field number %d has type %T, expected *bool",
			num, desc.ExtensionType)
	}
	return nil
}
