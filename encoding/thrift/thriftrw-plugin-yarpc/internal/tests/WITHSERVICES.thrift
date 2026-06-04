// Thrift file that exercises every code path of the auth.actor_uuid
// annotation: optional and required struct fields, a flat method
// argument, a struct-typed argument whose own field carries the
// annotation, and a typedef-of-string argument. Its sibling
// NOSERVICES.thrift covers the no-service case; keeping this fixture
// separate lets TestCodeIsUpToDate enforce drift on the service-arg
// path too.

typedef string ActorIdentifier

struct Struct {
    1: optional string baz
    2: optional string UserIdentifier (auth.actor_uuid = "true")
}

struct StructRequiredUUID {
    1: optional string baz
    2: required string UserIdentifier (auth.actor_uuid = "true")
}

service TestService {
    // testMethod carries the annotation directly on a primitive arg.
    string testMethod(
        1: string notInterested,
        2: string interested (auth.actor_uuid = "true"),
    )

    // testStructMethod carries the annotation one struct hop away:
    // the arg is a Struct whose UserIdentifier field is annotated.
    // The generated args accessor must chain through
    // GetRequest().ActorUUID() to surface the UUID.
    string testStructMethod(
        1: Struct request,
    )

    // testTypedefMethod's arg is a `typedef string` whose getter
    // returns ActorIdentifier rather than string; the generated body
    // must wrap the call in string(...) to compile.
    string testTypedefMethod(
        1: ActorIdentifier identifier (auth.actor_uuid = "true"),
    )
}
