include "./common.thrift"

exception KeyDoesNotExist {
    1: optional string key
} (
    rpc.code = "invalid-argument"
)

exception IntegerMismatchError {
    1: required i64 expectedValue
    2: required i64 gotValue
} (
    rpc.code = "invalid-argument"
)

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


// This struct intentionally has the same shape as the `CompareAndSwap` wrapper
// `Store_CompareAndSwap_Args`, except all fields are optional.

// We use this to generate an invalid payload for testing.
struct OptionalCompareAndSwapWrapper {
    1: optional OptionalCompareAndSwap cas
}

struct OptionalCompareAndSwap {
    1: optional string key
    2: optional i64 currentValue
    3: optional i64 newValue
}
