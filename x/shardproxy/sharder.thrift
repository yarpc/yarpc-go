struct ShardInfoRequest {}

struct ShardInfoResponse {
    1: optional string identifier
    2: optional list<string> supportedShards
}

service Shard {
    ShardInfoResponse shardInfo(1: ShardInfoRequest r)
}
