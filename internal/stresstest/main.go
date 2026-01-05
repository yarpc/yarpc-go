// Copyright (c) 2026 Uber Technologies, Inc.
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

// Package main runs a stress test on each peer list implementation,
// concurrently adding, removing, connecting, disconnecting, and choosing
// peers.
//
// Output:
//
//	name                     stress  workers  min       max         mean       choices  updates
//	round-robin              true    1        245       8770915     146851     5253     46150
//	round-robin              false   1        207       418011      291        971224   1
//	round-robin              true    10       233       16849329    387306     19074    50669
//	round-robin              false   10       223       1578697     5346       1194757  1
//	round-robin              true    100      230       27465221    908544     75258    46810
//	round-robin              false   100      226       7461220     67418      1032094  1
//	round-robin              true    1000     233       22398858    2024969    330804   4576
//	round-robin              false   1000     229       21501281    802412     839121   1
//	round-robin              true    10000    233       92130003    19808992   346389   297
//	round-robin              false   10000    276       64225586    21145554   314307   1
//	round-robin              true    100000   230       938109206   206172150  420222   8
//	round-robin              false   100000   234       752404056   162852049  555573   1
//	random                   true    1        290       9613482     147458     5411     51211
//	random                   false   1        218       4010623     320        945794   1
//	random                   true    10       251       13586138    292117     25511    51602
//	random                   false   10       234       6359156     6460       945427   1
//	random                   true    100      245       22698254    635979     106915   41375
//	random                   false   100      245       11036988    90960      740277   1
//	random                   true    1000     248       16201139    1991433    335656   8817
//	random                   false   1000     241       20870812    1309373    510240   1
//	random                   true    10000    289       78923469    23178028   288479   125
//	random                   false   10000    297       80162200    24015477   278955   1
//	random                   true    100000   241       765837860   237899834  351979   25
//	random                   false   100000   289       694941549   212820591  365262   1
//	fewest-pending-requests  true    1        390       4419885     224512     3200     45697
//	fewest-pending-requests  false   1        313       1388333     437        811476   1
//	fewest-pending-requests  true    10       341       15878684    238641     29821    38477
//	fewest-pending-requests  false   10       328       5097597     8860       719393   1
//	fewest-pending-requests  true    100      338       12701474    452931     150094   35696
//	fewest-pending-requests  false   100      339       9182151     96564      705745   1
//	fewest-pending-requests  true    1000     329       13381772    2351399    285263   1816
//	fewest-pending-requests  false   1000     312       16021295    1546288    430947   1
//	fewest-pending-requests  true    10000    421       85027409    29813414   227209   135
//	fewest-pending-requests  false   10000    404       86757606    27587847   244877   1
//	fewest-pending-requests  true    100000   381       969925927   278125286  320719   11
//	fewest-pending-requests  false   100000   342       759484399   256068224  351255   1
//	two-random-choices       true    1        296       12530130    167011     4873     56520
//	two-random-choices       false   1        234       2157414     331        973400   1
//	two-random-choices       true    10       257       14773053    264627     27980    49152
//	two-random-choices       false   10       254       6038348     6769       896344   1
//	two-random-choices       true    100      264       16509807    566482     120169   47062
//	two-random-choices       false   100      265       8954712     78887      840535   1
//	two-random-choices       true    1000     262       16153194    1959869    341938   6304
//	two-random-choices       false   1000     257       14265730    1299014    512357   1
//	two-random-choices       true    10000    312       65980563    25155737   263431   75
//	two-random-choices       false   10000    315       68347456    23842068   282351   1
//	two-random-choices       true    100000   259       1164075075  114441088  663750   31
//	two-random-choices       false   100000   269       836529911   164734573  545225   1
package main

import (
	"fmt"
	"time"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/pendingheap"
	"go.uber.org/yarpc/peer/randpeer"
	"go.uber.org/yarpc/peer/roundrobin"
	"go.uber.org/yarpc/peer/tworandomchoices"
	"go.uber.org/yarpc/yarpctest"
)

func main() {
	fmt.Printf("name,stress,workers,min,max,mean,choices,updates\n")
	log := logger{}
	for _, c := range []struct {
		name    string
		newFunc func(peer.Transport) peer.ChooserList
	}{
		{
			name: "round-robin",
			newFunc: func(trans peer.Transport) peer.ChooserList {
				return roundrobin.New(trans)
			},
		},
		{
			name: "random",
			newFunc: func(trans peer.Transport) peer.ChooserList {
				return randpeer.New(trans)
			},
		},
		{
			name: "fewest-pending-requests",
			newFunc: func(trans peer.Transport) peer.ChooserList {
				return pendingheap.New(trans)
			},
		},
		{
			name: "two-random-choices",
			newFunc: func(trans peer.Transport) peer.ChooserList {
				return tworandomchoices.New(trans)
			},
		},
	} {
		for i := 1; i <= 1000; i *= 10 {
			for _, lowStress := range []bool{false, true} {
				report := yarpctest.ListStressTest{
					Workers:   i,
					Duration:  time.Second,
					Timeout:   time.Millisecond * time.Duration(i),
					LowStress: lowStress,
					New:       c.newFunc,
				}.Run(log)
				if report.Choices != 0 {
					fmt.Printf("%s,%v,%d,%d,%d,%d,%d,%d\n",
						c.name,
						!lowStress,
						uint64(i),
						uint64(report.Min),
						uint64(report.Max),
						uint64(report.Total/time.Duration(report.Choices)),
						report.Choices,
						report.Updates,
					)
				}
			}
		}
	}
}

type logger struct{}

func (logger) Logf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}
