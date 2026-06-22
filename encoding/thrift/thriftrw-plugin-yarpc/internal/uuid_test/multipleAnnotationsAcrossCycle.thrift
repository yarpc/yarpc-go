struct CycleNode {
    1: optional CycleNode loop
    2: optional string idInsideCycle (auth.actor_uuid = "true")
}

service TestService {
    string testMethod(
        1: CycleNode arg,
        2: string idOutside (auth.actor_uuid = "true"),
    )
}
