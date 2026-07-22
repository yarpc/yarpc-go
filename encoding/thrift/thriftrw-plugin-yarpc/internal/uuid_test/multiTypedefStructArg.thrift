// Two-hop typedef chain pointing at the same underlying struct.
// thriftrw emits both AliasedInner and DoubleAliasedInner as
// distinct named Go types, but Go's pointer-conversion rules allow
// (*Inner)(*DoubleAliasedInner) directly because all three types
// share the same underlying struct definition.
struct Inner {
    1: optional string id (auth.actor_uuid = "true")
}

typedef Inner AliasedInner
typedef AliasedInner DoubleAliasedInner

struct OuterContainer {
    1: optional DoubleAliasedInner deeplyAliased
}

service TestService {
    string testMethod(
        1: OuterContainer outer,
    )
}
