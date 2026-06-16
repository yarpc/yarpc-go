// Thrift file that exercises every code path of the auth.actor_uuid
// annotation: optional and required struct fields, a flat method
// argument, a struct-typed argument whose own field carries the
// annotation, a typedef-of-string argument, and an arg whose path to
// the annotation runs through several nested struct hops. Its sibling
// NOSERVICES.thrift covers the no-service case; keeping this fixture
// separate lets TestCodeIsUpToDate enforce drift on the service-arg
// path too.

typedef string ActorIdentifier

struct Struct {
    1: optional string baz
    2: optional string UserIdentifier (auth.actor_uuid = "true")
}

struct StructRequiredUUID {
    1: optional string baz
    2: required string UserIdentifier (auth.actor_uuid = "true")
}

// CycleA and CycleB form a mutually recursive struct cycle. ThriftRW
// happily compiles cyclic struct references (the generated Go uses
// pointer fields, e.g. `Peer *CycleB`, so the cycle is representable),
// even though no value of the type can ever be encoded end to end.
// They live here as a reference for the YARPC plugin's path walker.
struct CycleA {
    1: required CycleB peer
}

struct CycleB {
    1: required CycleA peer
}

// InnerLevel, MidLevel and OuterLevel form a three-deep struct chain
// used by testNestedMethod to exercise the arbitrary-depth walker:
// only the leaf field carries the annotation.
struct InnerLevel {
    1: optional string innerUUID (auth.actor_uuid = "true")
}

struct MidLevel {
    1: optional InnerLevel inner
}

struct OuterLevel {
    1: optional MidLevel mid
}

// AliasedInner / DoubleAliasedInner / OuterWithAlias exercise descent
// through typedef-of-struct hops: thriftrw emits AliasedInner as a
// distinct named Go type (`type AliasedInner Inner`), so the chain
// must cast through Inner to reach GetXxx() accessors. The two-hop
// alias confirms the cast stays valid even after multiple typedefs
// because `*DoubleAliasedInner`, `*AliasedInner` and `*Inner` all
// share the same underlying struct type.
struct Inner {
    1: optional string innerUUID (auth.actor_uuid = "true")
}

typedef Inner AliasedInner
typedef AliasedInner DoubleAliasedInner

struct OuterWithAlias {
    1: optional AliasedInner inner
    2: optional DoubleAliasedInner deeplyAliased
}

service TestService {
    // testMethod carries the annotation directly on a primitive arg.
    string testMethod(
        1: string notInterested,
        2: string interested (auth.actor_uuid = "true"),
    )

    // testStructMethod carries the annotation one struct hop away:
    // the arg is a Struct whose UserIdentifier field is annotated.
    // The generated args accessor must chain through
    // GetRequest().GetUserIdentifier() to surface the UUID.
    string testStructMethod(
        1: Struct request,
    )

    // testTypedefMethod's arg is a `typedef string` whose getter
    // returns ActorIdentifier rather than string; the generated body
    // must wrap the call in string(...) to compile.
    string testTypedefMethod(
        1: ActorIdentifier identifier (auth.actor_uuid = "true"),
    )

    // testNestedMethod's arg traverses three struct hops down to the
    // annotated leaf. The generated args accessor must walk all the
    // way down in a single chain:
    // t.GetNested().GetMid().GetInner().GetInnerUUID().
    string testNestedMethod(
        1: OuterLevel nested,
    )

    // testTypedefStructMethod descends through a typedef-of-struct
    // arg directly to the annotated leaf. thriftrw emits
    // GetTopLevel() returning *AliasedInner, on which GetInnerUUID()
    // does not exist; the generated body must cast through *Inner
    // first: (*Inner)(t.GetTopLevel()).GetInnerUUID().
    string testTypedefStructMethod(
        1: AliasedInner topLevel,
    )

    // testNestedTypedefStructMethod descends through a struct that
    // owns two typedef-of-struct hops (single- and double-aliased
    // typedefs to the same underlying struct). Each cast wraps the
    // partial chain so the next GetXxx() resolves; with two hops the
    // walker still picks the first reachable annotation by Thrift
    // field order, which is the inner field on OuterWithAlias.
    string testNestedTypedefStructMethod(
        1: OuterWithAlias outer,
    )

    // testDoubleTypedefStructMethod's arg is the two-hop typedef
    // DoubleAliasedInner directly, with no struct hop in between.
    // It pins the multi-hop case end-to-end: the walker resolves
    // through both typedef layers and the chain emits a single
    // (*Inner)(t.GetArg()) conversion. That single cast is legal
    // even though the static return type of GetArg() is
    // *DoubleAliasedInner, because Go's pointer-conversion rule is
    // transitively closed for typedef chains that share the same
    // underlying struct definition.
    string testDoubleTypedefStructMethod(
        1: DoubleAliasedInner arg,
    )
}
