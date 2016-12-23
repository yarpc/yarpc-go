service Hello {
    EchoResponse echo(1:EchoRequest echo)
}

struct EchoRequest {
    1: required string message;
    2: required i16 count;
}

struct EchoResponse {
    1: required string message;
    2: required i16 count;
}
