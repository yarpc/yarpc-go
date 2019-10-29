package main

import (
	"bytes"
	"os"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"go.uber.org/yarpc/x/ydump/thrift"

	"go.uber.org/thriftrw/compile"
	"go.uber.org/thriftrw/protocol"
	"gopkg.in/yaml.v2"
)

var (
	_thriftFile = flag.String("t", "", "Thrift file with the type")
	_thriftName = flag.String("symbol", "", "Thrift symbol name (can be function.arg, or function.res)")

	_serialze = flag.Bool("serialize", false, "Whether to serialize or deserialize a value")
)

func main() {
	flag.Parse()

	module, err := compile.Compile(*_thriftFile)
	if err != nil {
		panic(err)
	}

	var spec compile.TypeSpec
	if strings.Contains(*_thriftName, "::") {
		// If the name contains "::",  then look for a service / method.
		spec, err = findMethodType(module, *_thriftName)
	} else {
		spec, err = module.LookupType(*_thriftName)
	}

	if err != nil {
		log.Fatal(err)
	}

	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("Failed to read stdin: %v", err)
	}

	var output []byte
	if *_serialze {
		output, err = serialize(spec, input)
	} else {
		output, err = deserialize(spec, input)
	}
	if err != nil {
		log.Fatalf("Faield to convert: %v", err)
	}

	if _, err := os.Stdout.Write(output); err != nil {
		log.Fatalf("Failed to write output: %v", err)
	}
}

func deserialize(spec compile.TypeSpec, contents []byte) ([]byte, error) {
	contentsReader := bytes.NewReader(contents)
	w, err := protocol.Binary.Decode(contentsReader, spec.TypeCode())
	if err != nil {
		return nil, fmt.Errorf("failed to read binary as thrift: %v", err)
	}

	v, err := thrift.FromWireValue(spec, w)
	if err != nil {
		return nil, fmt.Errorf("failed to convert thrift to specified type: %v", err)
	}

	return yaml.Marshal(v)
}

func serialize(spec compile.TypeSpec, contents []byte) ([]byte, error) {
	var req map[string]interface{}
	if err := yaml.Unmarshal(contents, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal contents: %v", err)
	}

	v, err := thrift.ToWireValue(spec, req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request to thrift: %v", err)
	}

	buf := &bytes.Buffer{}
	if err := protocol.Binary.Encode(v, buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func findMethodType(module *compile.Module, name string) (compile.TypeSpec, error) {
	nameSplit := strings.Split(name, "::")
	if len(nameSplit) != 2 {
		return nil, fmt.Errorf("must specify Service::method as method name")
	}

	svcName := nameSplit[0]
	methodSplit := strings.Split(nameSplit[1], ".")
	if len(methodSplit) != 2 {
		return nil, fmt.Errorf("must specify whether you want arg or res, using Service::method.arg or Service::method.res")
	}

	methodName := methodSplit[0]
	svc, ok := module.Services[svcName]
	if !ok {
		return nil, fmt.Errorf("no such service %q", svcName)
	}

	method, ok := svc.Functions[methodName]
	if !ok {
		return nil, fmt.Errorf("no such method %q", methodName)
	}

	if methodSplit[1] == "res" {
		return method.ResultSpec.ReturnType, nil
	}

	return &compile.StructSpec{
		Name:   name,
		Fields: compile.FieldGroup(method.ArgsSpec),
	}, nil
}
