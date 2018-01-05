service HelloBinary {
    EchoBinaryResponse echo(1:EchoBinaryRequest echo)
}

struct EchoBinaryRequest {
    1: required binary message;
    2: required i16 count;
}

struct EchoBinaryResponse {
    1: required binary message;
    2: required i16 count;
}
