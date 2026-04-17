// Thrift file with no service to ensure that types_yarpc.go is always
// generated.

exception ExWithAnnotation {
    1: optional string foo
} (
    rpc.code = "OUT_OF_RANGE"
)

exception ExWithoutAnnotation {
    1: optional string bar
}

struct Struct {
    1: optional string baz
    2: optional string uuid (auth.actor_uuid = "true")
}
