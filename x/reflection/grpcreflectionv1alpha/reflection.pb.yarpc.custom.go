// Copyright (c) 2025 Uber Technologies, Inc.
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

package grpcreflectionv1alpha

// YARPCReflectionFileDescriptors returns an array of encoded filedescriptor
// for yaprc to use. Normally the filedescriptors are accessed through the
// reflection.ServerMeta that is injected into the container. The injection is
// happens using New{}YARPCProcedures to automatically inject for all services
// using the fx pattern.
// This will not work for the reflection service due to a chicken and egg
// problem: we need a server to access the reflection.Meta and to create a
// server we need access to all the reflection.Meta (including our own).
// We could use throwaway instantiation of the reflection service to access
// its meta, but this would require using the interface{} response of
// New{}YARPCProcedures meant for fx. Instead of being type unsafe, here we
// augment the generated code to get compile time safe access to the required
// filedescriptor
//
// After regeneration with the yarpc plugin, update this reference
var YARPCReflectionFileDescriptors = yarpcFileDescriptorClosure42a8ac412db3cb03
