// Code generated by thriftrw v1.2.0
// @generated

package common

import "go.uber.org/thriftrw/thriftreflect"

var ThriftModule = &thriftreflect.ThriftModule{Name: "common", Package: "go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/common", FilePath: "common.thrift", SHA1: "1bd2b34a2289d2767d66eff00fa74778a14a625f", Raw: rawIDL}

const rawIDL = "service EmptyService {}\n\nservice ExtendEmpty extends EmptyService {\n    void hello()\n}\n\nservice BaseService {\n    bool healthy()\n}\n\nservice ExtendOnly extends BaseService {\n    // A service without any functions except inherited ones\n}\n"
