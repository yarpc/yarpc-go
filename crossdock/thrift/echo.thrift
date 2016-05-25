// Note that type definitions are being declared before the service
// because Apache Thrift doesn't support forward references. ThriftRW
// works just fine with the service defined up top, but we're generating
// shapes for both libraries from this file.

struct Ping {
    1: required string beep
}

struct Pong {
    1: required string boop
}

service Echo {
    Pong echo(1: Ping ping) (
        ttlms = '100'
    )
}
