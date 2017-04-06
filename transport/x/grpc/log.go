// Copyright (c) 2017 Uber Technologies, Inc.
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

package grpc

import (
	"go.uber.org/zap"
	"google.golang.org/grpc/grpclog"
)

// SetLogger will set the given logger to be the logger for grpclog.
func SetLogger(logger *zap.Logger) {
	grpclog.SetLogger(newLogger(logger))
}

type logger struct {
	*zap.SugaredLogger
}

func newLogger(l *zap.Logger) *logger {
	return &logger{l.Sugar()}
}

func (l *logger) Print(args ...interface{}) {
	l.SugaredLogger.Info(args...)
}

func (l *logger) Printf(format string, args ...interface{}) {
	l.SugaredLogger.Infof(format, args...)
}

func (l *logger) Println(args ...interface{}) {
	l.SugaredLogger.Info(args...)
}

func (l *logger) Fatalln(args ...interface{}) {
	l.SugaredLogger.Fatal(args...)
}
