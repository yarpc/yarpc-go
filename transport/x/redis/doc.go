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

// Package redis provides an simple, EXPERIMENTAL queuing transport backed by
// a redis list. Use at your own risk.
//
// Current behavior (expected to change):
//  - the outbound uses `LPUSH` to place an RPC at rest onto the redis list
//    that's acting as a queue
//  - the inbound uses the atomic `BRPOPLPUSH` operation to dequeue items and
//    place them in a processing list
//  - processing failure/success cause a permanent removal of an item
//
// Sample usage:
//
//   client-side:
//      redisOutbound := redis.NewOutbound(
//          redis.NewRedis5Client(redisAddr),
//          "my-queue-key",   // where to enqueue items
//      )
//      ...
//      dispatcher := yarpc.NewDispatcher(Config{
//                      ...
//                      Outbounds: yarpc.Outbounds{
//                        "some-service": { Oneway: redisOutbound },
//                      }
//                  })
//
//
//   server-side:
//      redisInbound := redis.NewInbound(
//          redis.NewRedis5Client(redisAddr),
//          "my-queue-key",       // where to dequeue items from
//          "my-processing-key",  // where to put items while processing
//          time.Second,          // wait for up to timeout, when reading queue
//      )
//      ...
//      dispatcher := yarpc.NewDispatcher(Config{
//                      Name: "some-service",
//                      Inbounds: yarpc.Inbounds{ redisInbound },
//                      ...
//                  })
//
// From here, standard Oneway RPCs made from the client to 'some-service' will
// be transported to the server through a Redis queue.
//
// USE OF THIS PACKAGE SHOULD BE FOR EXPERIMENTAL PURPOSES ONLY.
// BEHAVIOR IS EXPECTED TO CHANGE.
package redis
