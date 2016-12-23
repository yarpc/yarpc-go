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
// 	client := json.New(clientConfig)
// 	var resBody GetValueResponse
// 	resMeta, err := client.Call(
// 		yarpc.NewReqMeta(ctx).Procedure("getValue"),
// 		&GetValueRequest{...},
// 		&resBody,
// 	)
//
// To register a JSON procedure, define functions in the format,
//
// 	f(ctx context.Context, body $reqBody) ($resBody, error)
//
// Where '$reqBody' and '$resBody' are either pointers to structs representing
// your request and response objects, or map[string]interface{}.
//
// Use the Procedure function to build procedures to register against a
// Router.
//
//  dispatcher.Register(json.Procedure("getValue", GetValue))
//  dispatcher.Register(json.Procedure("setValue", SetValue))
//
// Similarly, to register a oneway JSON procedure, define functions in the
// format,
//
// 	f(ctx context.Context, body $reqBody) error
//
// Where $reqBody is a map[string]interface{} or pointer to a struct.
//
// Use the OnewayProcedure function to build procedures to register against a
// Router.
//
//  dispatcher.Register(json.OnewayProcedure("setValue", SetValue))
//  dispatcher.Register(json.OnewayProcedure("runTask", RunTask))
//
package json
