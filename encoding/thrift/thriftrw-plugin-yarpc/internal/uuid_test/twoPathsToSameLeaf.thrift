struct Inner {
    1: optional string id (auth.actor_uuid = "true")
}

typedef Inner AliasedInner

struct OuterContainer {
    1: optional Inner first
    2: optional AliasedInner second
}

service TestService {
    string testMethod(
        1: OuterContainer outer,
    )
}
