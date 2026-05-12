struct RequiredStruct {
    1: optional string baz
    2: required string UserIdentifier (auth.actor_uuid = "true")
}
