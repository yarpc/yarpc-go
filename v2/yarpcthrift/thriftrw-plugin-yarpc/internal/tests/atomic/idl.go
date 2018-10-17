// Code generated by thriftrw v1.12.0. DO NOT EDIT.
// @generated

package atomic

import (
	"go.uber.org/thriftrw/thriftreflect"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/common"
)

// ThriftModule represents the IDL file used to generate this package.
var ThriftModule = &thriftreflect.ThriftModule{
	Name:     "atomic",
	Package:  "go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic",
	FilePath: "atomic.thrift",
	SHA1:     "2bc145384bb2472af911bf797c7c23331111638a",
	Includes: []*thriftreflect.ThriftModule{
		common.ThriftModule,
	},
	Raw: rawIDL,
}

const rawIDL = "include \"./common.thrift\"\n\nexception KeyDoesNotExist {\n    1: optional string key\n}\n\nexception IntegerMismatchError {\n    1: required i64 expectedValue\n    2: required i64 gotValue\n}\n\nstruct CompareAndSwap {\n    1: required string key\n    2: required i64 currentValue\n    3: required i64 newValue\n}\n\nservice ReadOnlyStore extends common.BaseService {\n    i64 integer(1: string key) throws (1: KeyDoesNotExist doesNotExist)\n}\n\nservice Store extends ReadOnlyStore {\n    void increment(1: string key, 2: i64 value)\n\n    void compareAndSwap(1: CompareAndSwap request)\n        throws (1: IntegerMismatchError mismatch)\n\n    oneway void forget(1: string key)\n}\n\n"