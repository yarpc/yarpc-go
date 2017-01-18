// Copyright (c) 2016 Uber Technologies, Inc.
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

package mocks

import cherami "github.com/uber/cherami-client-go/client/cherami"
import mock "github.com/stretchr/testify/mock"
import xcherami "go.uber.org/yarpc/transport/x/cherami"

// CheramiFactory is an autogenerated mock type for the CheramiFactory type
type CheramiFactory struct {
	mock.Mock
}

// GetClientWithFrontEnd provides a mock function with given fields: ip, port
func (_m *CheramiFactory) GetClientWithFrontEnd(ip string, port int) (cherami.Client, error) {
	ret := _m.Called(ip, port)

	var r0 cherami.Client
	if rf, ok := ret.Get(0).(func(string, int) cherami.Client); ok {
		r0 = rf(ip, port)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(cherami.Client)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, int) error); ok {
		r1 = rf(ip, port)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetClientWithHyperbahn provides a mock function with given fields:
func (_m *CheramiFactory) GetClientWithHyperbahn() (cherami.Client, error) {
	ret := _m.Called()

	var r0 cherami.Client
	if rf, ok := ret.Get(0).(func() cherami.Client); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(cherami.Client)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetConsumer provides a mock function with given fields: client, destination, consumerGroup, prefetchCount, timeoutInSec
func (_m *CheramiFactory) GetConsumer(client cherami.Client, destination string, consumerGroup string, prefetchCount int, timeoutInSec int) (cherami.Consumer, chan cherami.Delivery, error) {
	ret := _m.Called(client, destination, consumerGroup, prefetchCount, timeoutInSec)

	var r0 cherami.Consumer
	if rf, ok := ret.Get(0).(func(cherami.Client, string, string, int, int) cherami.Consumer); ok {
		r0 = rf(client, destination, consumerGroup, prefetchCount, timeoutInSec)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(cherami.Consumer)
		}
	}

	var r1 chan cherami.Delivery
	if rf, ok := ret.Get(1).(func(cherami.Client, string, string, int, int) chan cherami.Delivery); ok {
		r1 = rf(client, destination, consumerGroup, prefetchCount, timeoutInSec)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(chan cherami.Delivery)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(cherami.Client, string, string, int, int) error); ok {
		r2 = rf(client, destination, consumerGroup, prefetchCount, timeoutInSec)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetPublisher provides a mock function with given fields: client, destination
func (_m *CheramiFactory) GetPublisher(client cherami.Client, destination string) (cherami.Publisher, error) {
	ret := _m.Called(client, destination)

	var r0 cherami.Publisher
	if rf, ok := ret.Get(0).(func(cherami.Client, string) cherami.Publisher); ok {
		r0 = rf(client, destination)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(cherami.Publisher)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(cherami.Client, string) error); ok {
		r1 = rf(client, destination)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

var _ xcherami.CheramiFactory = (*CheramiFactory)(nil)
