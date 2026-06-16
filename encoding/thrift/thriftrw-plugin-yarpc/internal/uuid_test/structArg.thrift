// Service-method parameter whose type is itself a struct carrying an
// auth.actor_uuid-annotated field.

struct Request {
    1: optional string baz
    2: required string UserIdentifier (auth.actor_uuid = "true")
}

service TestService {
    string testMethod(
        1: Request request,
    )
}
