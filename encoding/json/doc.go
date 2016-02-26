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

// Package json provides the JSON encoding for YARPC.
//
// To make outbound requests using this encoding,
//
// 	client := json.New(outbound)
// 	var response GetValueResponse
// 	resBody, response, err := client.Call(
// 		&json.Request{
// 			Context: ctx,
// 			Procedure: "getValue",
// 		},
// 		&GetValueRequest{...},
// 		&response,
// 	)
//
// To register a JSON procedure, define functions in the format,
//
// 	f(req *json.Request, body $reqBody) ($resBody, *json.Response, error)
//
// Where '$reqBody' and '$resBody' are either pointers to structs representing
// your request and response objects, or map[string]interface{}.
//
// Use the Register and Procedure functions to register the procedures with a
// Registry.
//
// 	json.Register(r, json.Procedure("getValue", GetValue))
// 	json.Register(r, json.Procedure("setValue", SetValue))
//
package json
