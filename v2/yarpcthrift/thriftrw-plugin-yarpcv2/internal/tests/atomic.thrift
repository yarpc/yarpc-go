include "./common.thrift"

exception KeyDoesNotExist {
    1: optional string key
}

exception IntegerMismatchError {
    1: required i64 expectedValue
    2: required i64 gotValue
}

struct CompareAndSwap {
    1: required string key
    2: required i64 currentValue
    3: required i64 newValue
}

service ReadOnlyStore extends common.BaseService {
    i64 integer(1: string key) throws (1: KeyDoesNotExist doesNotExist)
}

service Store extends ReadOnlyStore {
    void increment(1: string key, 2: i64 value)

    void compareAndSwap(1: CompareAndSwap request)
        throws (1: IntegerMismatchError mismatch)

    oneway void forget(1: string key)
}

