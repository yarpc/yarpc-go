// CycleNode self-references both directly (`direct`) and through a
// typedef alias (`loop`).

struct CycleNode {
    1: required AliasedNode loop
    2: optional CycleNode direct
    3: optional string id (auth.actor_uuid = "true")
}

typedef CycleNode AliasedNode

service TestService {
    string testMethod(
        1: CycleNode arg,
    )
}
