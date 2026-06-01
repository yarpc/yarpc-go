
struct RedStruct {
    1: optional string baz
    2: optional string UserIdentifier (auth.actor_uuid = "true")
}

struct GreenStruct {
    1: optional string baz
    2: optional string CatIdentifier (auth.actor_uuid = "true")
}
