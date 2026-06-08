// Method args reach the auth.actor_uuid annotation through two
// distinct struct-typed args, each one struct hop in. The Go *_Args
// type can only carry one ActorUUID() accessor, so
// validateUUIDAnnotations must reject this even though both inner
// structs individually carry exactly one annotation.
struct Alpha {
    1: required string alphaID (auth.actor_uuid = "true")
}

struct Beta {
    1: required string betaID (auth.actor_uuid = "true")
}

service TestService {
    string testMethod(
        1: Alpha first,
        2: Beta  second,
    )
}
