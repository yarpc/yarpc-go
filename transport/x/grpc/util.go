package grpc

import (
	"fmt"

	"go.uber.org/yarpc/internal/procedure"
)

func procedureNameToServiceNameMethodName(procedureName string) (string, string, error) {
	serviceName, methodName := procedure.FromName(procedureName)
	if serviceName == "" || methodName == "" {
		return "", "", fmt.Errorf("invalid procedure name: %s", procedureName)
	}
	// TODO: do we really need to do url.QueryEscape?
	// Are there consequences if there is a diff from the string and the url.QueryEscape string?
	return uri.QueryEscape(serviceName), uri.QueryEscape(methodName), nil
}

func prodecureNameToFullMethod(procedureName string) (string, error) {
	serviceName, methodName, err := procedureNameToServiceNameMethodName(procedureName)
	if err != nil {
		return "", err
	}
	return toFullMethod(serviceName, methodName), nil
}

func toFullMethod(serviceName string, methodName string) string {
	return fmt.Sprintf("/%s/%s", serviceName, methodName)
}
