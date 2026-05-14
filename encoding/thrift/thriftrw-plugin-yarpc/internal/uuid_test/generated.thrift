struct Struct {
    1: optional string baz
    2: optional string UserIdentifier (auth.actor_uuid = "true")
}

struct StructRequiredUUID {
    1: optional string baz
    2: required string UserIdentifier (auth.actor_uuid = "true")
}

service TestService {
    string testMethod(
        1: string notInterested,
        2: string interested (auth.actor_uuid = "true"),
    )
}