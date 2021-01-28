package thrift_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/thrift/internal/observabilitytest/test"
	"go.uber.org/yarpc/encoding/thrift/internal/observabilitytest/test/testserviceclient"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/yarpc/yarpctest"
)

type testStructure struct {
	name   string
	req    *yarpctest.Call
	expReq map[string]string
}

var allTests map[string]testStructure

func validateReq(testname string, ctx context.Context) (bool, string) {
	test := allTests[testname]
	call := yarpc.CallFromContext(ctx)
	for name, value := range test.expReq {
		switch name {
		case "CallerProcedure":
			if call.CallerProcedure() != value {
				err := "TestName(" + testname + ") - CallerProcedure '" + call.CallerProcedure() + "' does not match with expected value '" + value + "'"
				return false, err
			}
		case "Procedure":
			if call.Procedure() != value {
				err := "TestName(" + testname + ") - Procedure '" + call.Procedure() + "' does not match with expected value '" + value + "'"
				return false, err
			}
		}
	}
	return true, ""
}

func runTest(t *testing.T, test testStructure, client testserviceclient.Interface, testName string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	ctx = yarpctest.ContextWithCall(ctx, test.req)
	defer cancel()

	_, err := client.Call(ctx, testName)
	require.NoError(t, err, "unexpected error")
}

func TestThriftMetrics1(t *testing.T) {
	transports := []string{tchannel.TransportName, http.TransportName}

	tests := []testStructure{
		{
			name: "test1",
			req: &yarpctest.Call{
				Procedure: "ABC1",
			},
			expReq: map[string]string{
				"CallerProcedure": "",
				"Procedure":       "TestService::Call",
			},
		},
	}
	allTests = make(map[string]testStructure)

	for _, trans := range transports {
		t.Run(trans+" thift call", func(t *testing.T) {
			client, _, _, _, cleanup := initClientAndServer(t, trans, testServer1{})
			defer cleanup()

			for _, test := range tests {
				testName := trans + "_" + test.name
				allTests[testName] = test
				runTest(t, test, client, testName)
			}
		})
	}
}

type testServer1 struct{}

func (testServer1) Call(ctx context.Context, val string) (string, error) {

	ok, err := validateReq(val, ctx)
	if ok == true {
		return val, nil
	}

	return "", &test.ExceptionWithoutCode{Val: err}
}
