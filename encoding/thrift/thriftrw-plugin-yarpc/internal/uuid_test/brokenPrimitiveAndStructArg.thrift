// Method args reach the auth.actor_uuid annotation through TWO args:
// one directly, and one via a struct hop. Each Go *_Args type can
// only carry one ActorUUID() accessor, so validateUUIDAnnotations
// must reject this — even though each individual struct (Inner) and
// each individual arg's own annotations map looks legal in isolation.
struct Inner {
    1: required string id (auth.actor_uuid = "true")
}

service TestService {
    string testMethod(
        1: string firstUUID (auth.actor_uuid = "true"),
        2: Inner  second,
    )
}
