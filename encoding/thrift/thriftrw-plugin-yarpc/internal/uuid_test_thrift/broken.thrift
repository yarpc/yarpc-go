
// We cannot have 2 fields annotated with actor uuid in the same struct.
struct BadStruct {
    1: optional string baz (auth.actor_uuid = "true")
    2: optional string UserIdentifier (auth.actor_uuid = "true")
}
