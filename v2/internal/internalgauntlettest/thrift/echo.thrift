struct EchoRequest {
  1: required string message
}

struct EchoResponse {
  1: required string message
}

service Echo{
  EchoResponse Echo(1: EchoRequest request)
}
