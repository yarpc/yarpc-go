service Hello {
    oneway void sink(1:SinkRequest snk)
}

struct SinkRequest {
    1: required string message;
    2: required i16 count;
}
