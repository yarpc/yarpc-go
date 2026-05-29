// Thrift file that exercises every code path of the auth.actor_uuid
// annotation: optional and required struct fields, plus a service method
// argument. Its sibling NOSERVICES.thrift covers the no-service case;
// keeping this fixture separate lets TestCodeIsUpToDate enforce drift on
// the service-arg path too.

struct Struct {
    1: optional string baz
    2: optional string UserIdentifier (auth.actor_uuid = "true")
}

struct StructRequiredUUID {
    1: optional string baz
    2: required string UserIdentifier (auth.actor_uuid = "true")
}

service TestService {
    string testMethod(
        1: string notInterested,
        2: string interested (auth.actor_uuid = "true"),
    )
}
