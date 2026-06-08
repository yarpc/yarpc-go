// Service-method parameter whose type is itself a struct carrying an
// auth.actor_uuid-annotated field. The outer arg has no annotation;
// uuidFieldInArgs must descend one struct hop to detect it, and return
// the OUTER arg (so the generated *_Args accessor lives on it) with
// .IsStruct=true so the template chains ".ActorUUID()".
struct Request {
    1: optional string baz
    2: required string UserIdentifier (auth.actor_uuid = "true")
}

service TestService {
    string testMethod(
        1: Request request,
    )
}
