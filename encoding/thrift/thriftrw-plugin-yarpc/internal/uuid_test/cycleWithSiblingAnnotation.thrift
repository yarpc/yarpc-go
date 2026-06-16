// CycleNode is a self-referencing struct: it has both a cyclic field
// (`loop`, of its own type) and a sibling field (`id`) carrying the
// annotation.

struct CycleNode {
    1: required CycleNode loop
    2: optional string id (auth.actor_uuid = "true")
}

service TestService {
    string testMethod(
        1: CycleNode arg,
    )
}
