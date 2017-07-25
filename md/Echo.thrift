namespace java io.yarpc.thrift.generated

service Echo {
    Pong echo(1: Ping ping) (
        ttlms = '100'
    )
}

struct Ping {
    1: required string beep
}

struct Pong {
    1: required string boop
}
