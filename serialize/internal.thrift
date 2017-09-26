struct RPC {
	1: required binary spanContext

	2: required string callerName
	3: required string serviceName
	4: required string encoding
	5: required string procedure

	6: optional map<string,string> headers
	7: optional string shardKey
	8: optional string routingKey
	9: optional string routingDelegate
	10: optional binary body
  11: optional Features features
}

struct Features {
  1: optional bool supportsBothResponseAndError
}
