// Copyright (c) 2022 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Autogenerated by Thrift Compiler (1.0.0-dev)
// DO NOT EDIT UNLESS YOU ARE SURE THAT YOU KNOW WHAT YOU ARE DOING

package gauntlet_tchannel

import (
	"bytes"
	"fmt"
	"github.com/uber/tchannel-go/thirdparty/github.com/apache/thrift/lib/go/thrift"
)

// (needed to ensure safety because of naive import list construction.)
var _ = thrift.ZERO
var _ = fmt.Printf
var _ = bytes.Equal

type SecondService interface { //Print 'testOneway(%d): Sleeping...' with secondsToSleep as '%d'
	//sleep 'secondsToSleep'
	//Print 'testOneway(%d): done sleeping!' with secondsToSleep as '%d'
	//@param i32 secondsToSleep - the number of seconds to sleep

	BlahBlah() (err error)
	// Prints 'testString("%s")' with thing as '%s'
	// @param string thing - the string to print
	// @return string - returns the string 'thing'
	//
	// Parameters:
	//  - Thing
	SecondtestString(thing string) (r string, err error)
}

// Print 'testOneway(%d): Sleeping...' with secondsToSleep as '%d'
// sleep 'secondsToSleep'
// Print 'testOneway(%d): done sleeping!' with secondsToSleep as '%d'
// @param i32 secondsToSleep - the number of seconds to sleep
type SecondServiceClient struct {
	Transport       thrift.TTransport
	ProtocolFactory thrift.TProtocolFactory
	InputProtocol   thrift.TProtocol
	OutputProtocol  thrift.TProtocol
	SeqId           int32
}

func NewSecondServiceClientFactory(t thrift.TTransport, f thrift.TProtocolFactory) *SecondServiceClient {
	return &SecondServiceClient{Transport: t,
		ProtocolFactory: f,
		InputProtocol:   f.GetProtocol(t),
		OutputProtocol:  f.GetProtocol(t),
		SeqId:           0,
	}
}

func NewSecondServiceClientProtocol(t thrift.TTransport, iprot thrift.TProtocol, oprot thrift.TProtocol) *SecondServiceClient {
	return &SecondServiceClient{Transport: t,
		ProtocolFactory: nil,
		InputProtocol:   iprot,
		OutputProtocol:  oprot,
		SeqId:           0,
	}
}

func (p *SecondServiceClient) BlahBlah() (err error) {
	if err = p.sendBlahBlah(); err != nil {
		return
	}
	return p.recvBlahBlah()
}

func (p *SecondServiceClient) sendBlahBlah() (err error) {
	oprot := p.OutputProtocol
	if oprot == nil {
		oprot = p.ProtocolFactory.GetProtocol(p.Transport)
		p.OutputProtocol = oprot
	}
	p.SeqId++
	if err = oprot.WriteMessageBegin("blahBlah", thrift.CALL, p.SeqId); err != nil {
		return
	}
	args := SecondServiceBlahBlahArgs{}
	if err = args.Write(oprot); err != nil {
		return
	}
	if err = oprot.WriteMessageEnd(); err != nil {
		return
	}
	return oprot.Flush()
}

func (p *SecondServiceClient) recvBlahBlah() (err error) {
	iprot := p.InputProtocol
	if iprot == nil {
		iprot = p.ProtocolFactory.GetProtocol(p.Transport)
		p.InputProtocol = iprot
	}
	method, mTypeId, seqId, err := iprot.ReadMessageBegin()
	if err != nil {
		return
	}
	if method != "blahBlah" {
		err = thrift.NewTApplicationException(thrift.WRONG_METHOD_NAME, "blahBlah failed: wrong method name")
		return
	}
	if p.SeqId != seqId {
		err = thrift.NewTApplicationException(thrift.BAD_SEQUENCE_ID, "blahBlah failed: out of sequence response")
		return
	}
	if mTypeId == thrift.EXCEPTION {
		error159 := thrift.NewTApplicationException(thrift.UNKNOWN_APPLICATION_EXCEPTION, "Unknown Exception")
		var error160 error
		error160, err = error159.Read(iprot)
		if err != nil {
			return
		}
		if err = iprot.ReadMessageEnd(); err != nil {
			return
		}
		err = error160
		return
	}
	if mTypeId != thrift.REPLY {
		err = thrift.NewTApplicationException(thrift.INVALID_MESSAGE_TYPE_EXCEPTION, "blahBlah failed: invalid message type")
		return
	}
	result := SecondServiceBlahBlahResult{}
	if err = result.Read(iprot); err != nil {
		return
	}
	if err = iprot.ReadMessageEnd(); err != nil {
		return
	}
	return
}

// Prints 'testString("%s")' with thing as '%s'
// @param string thing - the string to print
// @return string - returns the string 'thing'
//
// Parameters:
//   - Thing
func (p *SecondServiceClient) SecondtestString(thing string) (r string, err error) {
	if err = p.sendSecondtestString(thing); err != nil {
		return
	}
	return p.recvSecondtestString()
}

func (p *SecondServiceClient) sendSecondtestString(thing string) (err error) {
	oprot := p.OutputProtocol
	if oprot == nil {
		oprot = p.ProtocolFactory.GetProtocol(p.Transport)
		p.OutputProtocol = oprot
	}
	p.SeqId++
	if err = oprot.WriteMessageBegin("secondtestString", thrift.CALL, p.SeqId); err != nil {
		return
	}
	args := SecondServiceSecondtestStringArgs{
		Thing: thing,
	}
	if err = args.Write(oprot); err != nil {
		return
	}
	if err = oprot.WriteMessageEnd(); err != nil {
		return
	}
	return oprot.Flush()
}

func (p *SecondServiceClient) recvSecondtestString() (value string, err error) {
	iprot := p.InputProtocol
	if iprot == nil {
		iprot = p.ProtocolFactory.GetProtocol(p.Transport)
		p.InputProtocol = iprot
	}
	method, mTypeId, seqId, err := iprot.ReadMessageBegin()
	if err != nil {
		return
	}
	if method != "secondtestString" {
		err = thrift.NewTApplicationException(thrift.WRONG_METHOD_NAME, "secondtestString failed: wrong method name")
		return
	}
	if p.SeqId != seqId {
		err = thrift.NewTApplicationException(thrift.BAD_SEQUENCE_ID, "secondtestString failed: out of sequence response")
		return
	}
	if mTypeId == thrift.EXCEPTION {
		error161 := thrift.NewTApplicationException(thrift.UNKNOWN_APPLICATION_EXCEPTION, "Unknown Exception")
		var error162 error
		error162, err = error161.Read(iprot)
		if err != nil {
			return
		}
		if err = iprot.ReadMessageEnd(); err != nil {
			return
		}
		err = error162
		return
	}
	if mTypeId != thrift.REPLY {
		err = thrift.NewTApplicationException(thrift.INVALID_MESSAGE_TYPE_EXCEPTION, "secondtestString failed: invalid message type")
		return
	}
	result := SecondServiceSecondtestStringResult{}
	if err = result.Read(iprot); err != nil {
		return
	}
	if err = iprot.ReadMessageEnd(); err != nil {
		return
	}
	value = result.GetSuccess()
	return
}

type SecondServiceProcessor struct {
	processorMap map[string]thrift.TProcessorFunction
	handler      SecondService
}

func (p *SecondServiceProcessor) AddToProcessorMap(key string, processor thrift.TProcessorFunction) {
	p.processorMap[key] = processor
}

func (p *SecondServiceProcessor) GetProcessorFunction(key string) (processor thrift.TProcessorFunction, ok bool) {
	processor, ok = p.processorMap[key]
	return processor, ok
}

func (p *SecondServiceProcessor) ProcessorMap() map[string]thrift.TProcessorFunction {
	return p.processorMap
}

func NewSecondServiceProcessor(handler SecondService) *SecondServiceProcessor {

	self163 := &SecondServiceProcessor{handler: handler, processorMap: make(map[string]thrift.TProcessorFunction)}
	self163.processorMap["blahBlah"] = &secondServiceProcessorBlahBlah{handler: handler}
	self163.processorMap["secondtestString"] = &secondServiceProcessorSecondtestString{handler: handler}
	return self163
}

func (p *SecondServiceProcessor) Process(iprot, oprot thrift.TProtocol) (success bool, err thrift.TException) {
	name, _, seqId, err := iprot.ReadMessageBegin()
	if err != nil {
		return false, err
	}
	if processor, ok := p.GetProcessorFunction(name); ok {
		return processor.Process(seqId, iprot, oprot)
	}
	iprot.Skip(thrift.STRUCT)
	iprot.ReadMessageEnd()
	x164 := thrift.NewTApplicationException(thrift.UNKNOWN_METHOD, "Unknown function "+name)
	oprot.WriteMessageBegin(name, thrift.EXCEPTION, seqId)
	x164.Write(oprot)
	oprot.WriteMessageEnd()
	oprot.Flush()
	return false, x164

}

type secondServiceProcessorBlahBlah struct {
	handler SecondService
}

func (p *secondServiceProcessorBlahBlah) Process(seqId int32, iprot, oprot thrift.TProtocol) (success bool, err thrift.TException) {
	args := SecondServiceBlahBlahArgs{}
	if err = args.Read(iprot); err != nil {
		iprot.ReadMessageEnd()
		x := thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error())
		oprot.WriteMessageBegin("blahBlah", thrift.EXCEPTION, seqId)
		x.Write(oprot)
		oprot.WriteMessageEnd()
		oprot.Flush()
		return false, err
	}

	iprot.ReadMessageEnd()
	result := SecondServiceBlahBlahResult{}
	var err2 error
	if err2 = p.handler.BlahBlah(); err2 != nil {
		x := thrift.NewTApplicationException(thrift.INTERNAL_ERROR, "Internal error processing blahBlah: "+err2.Error())
		oprot.WriteMessageBegin("blahBlah", thrift.EXCEPTION, seqId)
		x.Write(oprot)
		oprot.WriteMessageEnd()
		oprot.Flush()
		return true, err2
	}
	if err2 = oprot.WriteMessageBegin("blahBlah", thrift.REPLY, seqId); err2 != nil {
		err = err2
	}
	if err2 = result.Write(oprot); err == nil && err2 != nil {
		err = err2
	}
	if err2 = oprot.WriteMessageEnd(); err == nil && err2 != nil {
		err = err2
	}
	if err2 = oprot.Flush(); err == nil && err2 != nil {
		err = err2
	}
	if err != nil {
		return
	}
	return true, err
}

type secondServiceProcessorSecondtestString struct {
	handler SecondService
}

func (p *secondServiceProcessorSecondtestString) Process(seqId int32, iprot, oprot thrift.TProtocol) (success bool, err thrift.TException) {
	args := SecondServiceSecondtestStringArgs{}
	if err = args.Read(iprot); err != nil {
		iprot.ReadMessageEnd()
		x := thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err.Error())
		oprot.WriteMessageBegin("secondtestString", thrift.EXCEPTION, seqId)
		x.Write(oprot)
		oprot.WriteMessageEnd()
		oprot.Flush()
		return false, err
	}

	iprot.ReadMessageEnd()
	result := SecondServiceSecondtestStringResult{}
	var retval string
	var err2 error
	if retval, err2 = p.handler.SecondtestString(args.Thing); err2 != nil {
		x := thrift.NewTApplicationException(thrift.INTERNAL_ERROR, "Internal error processing secondtestString: "+err2.Error())
		oprot.WriteMessageBegin("secondtestString", thrift.EXCEPTION, seqId)
		x.Write(oprot)
		oprot.WriteMessageEnd()
		oprot.Flush()
		return true, err2
	} else {
		result.Success = &retval
	}
	if err2 = oprot.WriteMessageBegin("secondtestString", thrift.REPLY, seqId); err2 != nil {
		err = err2
	}
	if err2 = result.Write(oprot); err == nil && err2 != nil {
		err = err2
	}
	if err2 = oprot.WriteMessageEnd(); err == nil && err2 != nil {
		err = err2
	}
	if err2 = oprot.Flush(); err == nil && err2 != nil {
		err = err2
	}
	if err != nil {
		return
	}
	return true, err
}

// HELPER FUNCTIONS AND STRUCTURES

type SecondServiceBlahBlahArgs struct {
}

func NewSecondServiceBlahBlahArgs() *SecondServiceBlahBlahArgs {
	return &SecondServiceBlahBlahArgs{}
}

func (p *SecondServiceBlahBlahArgs) Read(iprot thrift.TProtocol) error {
	if _, err := iprot.ReadStructBegin(); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T read error: ", p), err)
	}

	for {
		_, fieldTypeId, fieldId, err := iprot.ReadFieldBegin()
		if err != nil {
			return thrift.PrependError(fmt.Sprintf("%T field %d read error: ", p, fieldId), err)
		}
		if fieldTypeId == thrift.STOP {
			break
		}
		if err := iprot.Skip(fieldTypeId); err != nil {
			return err
		}
		if err := iprot.ReadFieldEnd(); err != nil {
			return err
		}
	}
	if err := iprot.ReadStructEnd(); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T read struct end error: ", p), err)
	}
	return nil
}

func (p *SecondServiceBlahBlahArgs) Write(oprot thrift.TProtocol) error {
	if err := oprot.WriteStructBegin("blahBlah_args"); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T write struct begin error: ", p), err)
	}
	if err := oprot.WriteFieldStop(); err != nil {
		return thrift.PrependError("write field stop error: ", err)
	}
	if err := oprot.WriteStructEnd(); err != nil {
		return thrift.PrependError("write struct stop error: ", err)
	}
	return nil
}

func (p *SecondServiceBlahBlahArgs) String() string {
	if p == nil {
		return "<nil>"
	}
	return fmt.Sprintf("SecondServiceBlahBlahArgs(%+v)", *p)
}

type SecondServiceBlahBlahResult struct {
}

func NewSecondServiceBlahBlahResult() *SecondServiceBlahBlahResult {
	return &SecondServiceBlahBlahResult{}
}

func (p *SecondServiceBlahBlahResult) Read(iprot thrift.TProtocol) error {
	if _, err := iprot.ReadStructBegin(); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T read error: ", p), err)
	}

	for {
		_, fieldTypeId, fieldId, err := iprot.ReadFieldBegin()
		if err != nil {
			return thrift.PrependError(fmt.Sprintf("%T field %d read error: ", p, fieldId), err)
		}
		if fieldTypeId == thrift.STOP {
			break
		}
		if err := iprot.Skip(fieldTypeId); err != nil {
			return err
		}
		if err := iprot.ReadFieldEnd(); err != nil {
			return err
		}
	}
	if err := iprot.ReadStructEnd(); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T read struct end error: ", p), err)
	}
	return nil
}

func (p *SecondServiceBlahBlahResult) Write(oprot thrift.TProtocol) error {
	if err := oprot.WriteStructBegin("blahBlah_result"); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T write struct begin error: ", p), err)
	}
	if err := oprot.WriteFieldStop(); err != nil {
		return thrift.PrependError("write field stop error: ", err)
	}
	if err := oprot.WriteStructEnd(); err != nil {
		return thrift.PrependError("write struct stop error: ", err)
	}
	return nil
}

func (p *SecondServiceBlahBlahResult) String() string {
	if p == nil {
		return "<nil>"
	}
	return fmt.Sprintf("SecondServiceBlahBlahResult(%+v)", *p)
}

// Attributes:
//   - Thing
type SecondServiceSecondtestStringArgs struct {
	Thing string `thrift:"thing,1" db:"thing" json:"thing"`
}

func NewSecondServiceSecondtestStringArgs() *SecondServiceSecondtestStringArgs {
	return &SecondServiceSecondtestStringArgs{}
}

func (p *SecondServiceSecondtestStringArgs) GetThing() string {
	return p.Thing
}
func (p *SecondServiceSecondtestStringArgs) Read(iprot thrift.TProtocol) error {
	if _, err := iprot.ReadStructBegin(); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T read error: ", p), err)
	}

	for {
		_, fieldTypeId, fieldId, err := iprot.ReadFieldBegin()
		if err != nil {
			return thrift.PrependError(fmt.Sprintf("%T field %d read error: ", p, fieldId), err)
		}
		if fieldTypeId == thrift.STOP {
			break
		}
		switch fieldId {
		case 1:
			if err := p.ReadField1(iprot); err != nil {
				return err
			}
		default:
			if err := iprot.Skip(fieldTypeId); err != nil {
				return err
			}
		}
		if err := iprot.ReadFieldEnd(); err != nil {
			return err
		}
	}
	if err := iprot.ReadStructEnd(); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T read struct end error: ", p), err)
	}
	return nil
}

func (p *SecondServiceSecondtestStringArgs) ReadField1(iprot thrift.TProtocol) error {
	if v, err := iprot.ReadString(); err != nil {
		return thrift.PrependError("error reading field 1: ", err)
	} else {
		p.Thing = v
	}
	return nil
}

func (p *SecondServiceSecondtestStringArgs) Write(oprot thrift.TProtocol) error {
	if err := oprot.WriteStructBegin("secondtestString_args"); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T write struct begin error: ", p), err)
	}
	if err := p.writeField1(oprot); err != nil {
		return err
	}
	if err := oprot.WriteFieldStop(); err != nil {
		return thrift.PrependError("write field stop error: ", err)
	}
	if err := oprot.WriteStructEnd(); err != nil {
		return thrift.PrependError("write struct stop error: ", err)
	}
	return nil
}

func (p *SecondServiceSecondtestStringArgs) writeField1(oprot thrift.TProtocol) (err error) {
	if err := oprot.WriteFieldBegin("thing", thrift.STRING, 1); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T write field begin error 1:thing: ", p), err)
	}
	if err := oprot.WriteString(string(p.Thing)); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T.thing (1) field write error: ", p), err)
	}
	if err := oprot.WriteFieldEnd(); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T write field end error 1:thing: ", p), err)
	}
	return err
}

func (p *SecondServiceSecondtestStringArgs) String() string {
	if p == nil {
		return "<nil>"
	}
	return fmt.Sprintf("SecondServiceSecondtestStringArgs(%+v)", *p)
}

// Attributes:
//   - Success
type SecondServiceSecondtestStringResult struct {
	Success *string `thrift:"success,0" db:"success" json:"success,omitempty"`
}

func NewSecondServiceSecondtestStringResult() *SecondServiceSecondtestStringResult {
	return &SecondServiceSecondtestStringResult{}
}

var SecondServiceSecondtestStringResult_Success_DEFAULT string

func (p *SecondServiceSecondtestStringResult) GetSuccess() string {
	if !p.IsSetSuccess() {
		return SecondServiceSecondtestStringResult_Success_DEFAULT
	}
	return *p.Success
}
func (p *SecondServiceSecondtestStringResult) IsSetSuccess() bool {
	return p.Success != nil
}

func (p *SecondServiceSecondtestStringResult) Read(iprot thrift.TProtocol) error {
	if _, err := iprot.ReadStructBegin(); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T read error: ", p), err)
	}

	for {
		_, fieldTypeId, fieldId, err := iprot.ReadFieldBegin()
		if err != nil {
			return thrift.PrependError(fmt.Sprintf("%T field %d read error: ", p, fieldId), err)
		}
		if fieldTypeId == thrift.STOP {
			break
		}
		switch fieldId {
		case 0:
			if err := p.ReadField0(iprot); err != nil {
				return err
			}
		default:
			if err := iprot.Skip(fieldTypeId); err != nil {
				return err
			}
		}
		if err := iprot.ReadFieldEnd(); err != nil {
			return err
		}
	}
	if err := iprot.ReadStructEnd(); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T read struct end error: ", p), err)
	}
	return nil
}

func (p *SecondServiceSecondtestStringResult) ReadField0(iprot thrift.TProtocol) error {
	if v, err := iprot.ReadString(); err != nil {
		return thrift.PrependError("error reading field 0: ", err)
	} else {
		p.Success = &v
	}
	return nil
}

func (p *SecondServiceSecondtestStringResult) Write(oprot thrift.TProtocol) error {
	if err := oprot.WriteStructBegin("secondtestString_result"); err != nil {
		return thrift.PrependError(fmt.Sprintf("%T write struct begin error: ", p), err)
	}
	if err := p.writeField0(oprot); err != nil {
		return err
	}
	if err := oprot.WriteFieldStop(); err != nil {
		return thrift.PrependError("write field stop error: ", err)
	}
	if err := oprot.WriteStructEnd(); err != nil {
		return thrift.PrependError("write struct stop error: ", err)
	}
	return nil
}

func (p *SecondServiceSecondtestStringResult) writeField0(oprot thrift.TProtocol) (err error) {
	if p.IsSetSuccess() {
		if err := oprot.WriteFieldBegin("success", thrift.STRING, 0); err != nil {
			return thrift.PrependError(fmt.Sprintf("%T write field begin error 0:success: ", p), err)
		}
		if err := oprot.WriteString(string(*p.Success)); err != nil {
			return thrift.PrependError(fmt.Sprintf("%T.success (0) field write error: ", p), err)
		}
		if err := oprot.WriteFieldEnd(); err != nil {
			return thrift.PrependError(fmt.Sprintf("%T write field end error 0:success: ", p), err)
		}
	}
	return err
}

func (p *SecondServiceSecondtestStringResult) String() string {
	if p == nil {
		return "<nil>"
	}
	return fmt.Sprintf("SecondServiceSecondtestStringResult(%+v)", *p)
}
