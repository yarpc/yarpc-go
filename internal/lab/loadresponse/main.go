// Copyright (c) 2020 Uber Technologies, Inc.
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

// This benchmark measures the maximum capacity of a cluster as it depends on
// the configuration of the cluster and the load balancing algorithm.
// This uses a real YARPC peer list and simulates the behavior of a network
// using channels and goroutines.
// Non-blocking writes to channels simulate network loss and buffer overload.
//
// The theoretical maximum throughput capacity of a cluster is the sum of the
// capacities of the individual workers.
// The actual capacity of a cluster falls short of this figure depending on the
// load distribution algorithm and the variation in capacities of individual
// workers.
//
// For example, when every individual client uses a random peer selection
// algorithm, the load distribution if binomial so the least loaded worker will
// tend to vary from the load of the most loaded worker and capacity to do work
// will be lost to inefficiency.
// A round robin load distribution is fair.
// An individual client does not favor fast workers over slow workers, although
// variance in load distribution tends to vanish with large numbers of clients.
// A fewest pending requests strategy, will tend to produce an egalitarian
// load.
//
// As the load on a cluster increases, with a particular number of clients,
// each with a number of workers sharing a peer list go generate concurrent
// requests at a particular total throughput, this simulation approaches the
// maximum capacity of the cluster.
// Beyond that point, the ratio of responses to requests will drop, perhaps
// precipitously if requests begin to time out in queue.
//
// The theory is that less even load distributions will result in less
// efficient clusters and a lower maximum throughput before congestion
// collapse.

package main

import (
	"flag"
	"fmt"
	"math/bits"
	"os"
	"time"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/pendingheap"
	"go.uber.org/yarpc/peer/randpeer"
	"go.uber.org/yarpc/peer/roundrobin"
	"go.uber.org/yarpc/peer/tworandomchoices"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func newRandom(trans peer.Transport) peer.ChooserList { return randpeer.New(trans) }
func newRR(trans peer.Transport) peer.ChooserList     { return roundrobin.New(trans) }
func newHeap(trans peer.Transport) peer.ChooserList   { return pendingheap.New(trans) }
func newTRC(trans peer.Transport) peer.ChooserList    { return tworandomchoices.New(trans) }

type listFunc struct {
	name    string
	newList func(peer.Transport) peer.ChooserList
}

type listFlags uint

const (
	randomFlag listFlags = 1 << iota
	roundRobinFlag
	fewestPendingRequestsFlag
	twoRandomChoicesFlag
	lastLbFlag
)

type listFlag struct {
	name  string
	flag  listFlags
	flags *listFlags
}

func (b *listFlag) String() string {
	return b.name
}

func (b *listFlag) IsBoolFlag() bool {
	return true
}

func (b *listFlag) Set(s string) error {
	*b.flags |= b.flag
	return nil
}

func run() error {
	var listFlags listFlags
	var listFuncs []listFunc
	var maxPace time.Duration
	var latency time.Duration
	var skew bool
	var single bool
	var x Experiment
	var requestCount int

	fs := flag.NewFlagSet("loader", flag.ExitOnError)

	fs.Var(&listFlag{name: "rr", flag: roundRobinFlag, flags: &listFlags}, "rr", "round robin")
	fs.Var(&listFlag{name: "random", flag: randomFlag, flags: &listFlags}, "random", "random")
	fs.Var(&listFlag{name: "fpr", flag: fewestPendingRequestsFlag, flags: &listFlags}, "fpr", "fewest pending requests")
	fs.Var(&listFlag{name: "trc", flag: twoRandomChoicesFlag, flags: &listFlags}, "trc", "two random choices")

	fs.IntVar(&x.ClientCount, "c", 2, "number of clients")
	fs.IntVar(&x.ClientConcurrency, "cc", 10, "number of workers per client")
	fs.DurationVar(&maxPace, "pace", 10*time.Millisecond, "mean time between requests for each client worker")
	fs.IntVar(&x.ServerCount, "s", 5, "number of servers")
	fs.IntVar(&x.ServerConcurrency, "sc", 10, "number of workers per server")
	fs.DurationVar(&latency, "latency", time.Millisecond, "mean time to handle a request for each server worker")
	fs.BoolVar(&skew, "skew", false, "whether to skew latency from 0⨉ to 2⨉ across the server cluster")
	fs.IntVar(&x.QueueLength, "q", 8, "queue length")
	fs.DurationVar(&x.Timeout, "ttl", time.Millisecond*4, "timeout")
	fs.IntVar(&requestCount, "requests", 1000, "request count")

	if err := fs.Parse(os.Args[1:]); err != nil || fs.NArg() != 0 {
		if err == flag.ErrHelp || fs.NArg() != 0 {
			fmt.Fprintf(os.Stderr, "usage: go run go.uber.org/yarpc/internal/lab/loadresponse [flags]\n\n")
			fs.PrintDefaults()
			return nil
		}
		return err
	}

	// None means all load balancing strategies.
	if listFlags == 0 {
		// All the ones.
		listFlags = lastLbFlag - 1
	} else if bits.OnesCount(uint(listFlags)) == 1 {
		single = true
	}

	if listFlags&randomFlag != 0 {
		listFuncs = append(listFuncs, listFunc{
			name:    "roundrobin",
			newList: newRR,
		})
	}
	if listFlags&roundRobinFlag != 0 {
		listFuncs = append(listFuncs, listFunc{
			name:    "random",
			newList: newRandom,
		})
	}
	if listFlags&fewestPendingRequestsFlag != 0 {
		listFuncs = append(listFuncs, listFunc{
			name:    "heap",
			newList: newHeap,
		})
	}
	if listFlags&twoRandomChoicesFlag != 0 {
		listFuncs = append(listFuncs, listFunc{
			name:    "tworandom",
			newList: newTRC,
		})
	}

	if skew && x.ServerCount > 1 {
		x.Latency = func(j, m int) time.Duration {
			// This series produces a sum of 2 with a mean value of 1 for any
			// server cluster size m.
			// 1: 0
			// 2: 0, 2/1
			// 3: 0, 2/2, 4/2
			// 4: 0, 2/3, 4/3, 6/3
			// 5: 0, 2/4, 4/4, 6/4, 8/4
			// And, as usual with integer ratios unlikely to overflow,
			// multiply, THEN divide.
			return time.Duration(j*2) * latency / time.Duration(m-1)
		}
	} else {
		x.Latency = func(j, m int) time.Duration {
			return latency
		}
	}

	if single {
		// headers
		fmt.Print("rps\tduration\tefficiency\trequests\tresponses\tdrops\ttimeouts\n")
		// rows
		for pace := time.Nanosecond; pace < time.Millisecond; pace = nextPace(pace, 10, 11) {
			x.Pace = pace
			x.Duration = time.Duration(requestCount) * pace / time.Duration(x.ClientCount*x.ClientConcurrency)
			for _, lf := range listFuncs {
				x.NewList = lf.newList
				results := x.Run()
				fmt.Printf(
					"%.3e\t%v\t%f\t%d\t%d\t%d\t%d",
					float64(time.Second)/float64(x.Pace)*float64(x.ClientCount*x.ClientConcurrency),
					results.Duration,
					float64(results.Responded)/float64(results.Requested),
					results.Requested,
					results.Responded,
					results.Dropped,
					results.TimedOut,
				)
			}
			fmt.Printf("\n")
		}

	} else { // multiple peer list types

		// headers
		fmt.Print("load")
		for _, lf := range listFuncs {
			fmt.Print("\t" + lf.name)
		}
		fmt.Print("\n")

		for pace := maxPace; pace > time.Microsecond; pace = nextPace(pace, 10, 11) {
			x.Pace = pace
			x.Duration = time.Duration(requestCount) * pace / time.Duration(x.ClientCount*x.ClientConcurrency)
			fmt.Printf("%.3e", float64(time.Second)/float64(x.Pace)*float64(x.ClientCount*x.ClientConcurrency))
			total := 0
			for _, lf := range listFuncs {
				x.NewList = lf.newList
				results := x.Run()
				fmt.Printf("\t%f", float64(results.Responded)/float64(results.Requested))
				total += results.Responded
			}
			fmt.Printf("\n")
			if total == 0 {
				break
			}
		}
	}

	return nil
}

func nextPace(pace time.Duration, mul int, div int) time.Duration {
	next := pace * time.Duration(mul) / time.Duration(div)
	if pace == next {
		if mul > div {
			return pace + time.Nanosecond
		}
		return pace - time.Nanosecond
	}
	return next
}
