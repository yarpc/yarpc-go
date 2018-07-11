// Copyright (c) 2018 Uber Technologies, Inc.
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

package chooserbenchmark

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	// display area size
	tabWidth      = 4
	displayWidth  = 120
	latencyHeight = 15
	latencyWidth  = 101

	// ascii code use to display histogram
	separator = "#"
	bar       = "*"

	// server request counter bucket count
	serverBucketCount = 10
)

// clientGroupMeta contains information to visualize a client group
type clientGroupMeta struct {
	// raw
	name  string
	count int

	// counter related metrics
	reqCount  int64
	resCount  int64
	histogram *Histogram

	// time related metrics
	rps         int
	meanLatency time.Duration
}

// serverGroupMeta contains information to visualize a server group
type serverGroupMeta struct {
	// raw
	name    string
	servers []*Server

	// counter related metrics
	requestCount int64
	histogram    *Histogram

	// time related metrics
	meanLatency time.Duration
}

// gauge contains settings to control display in terminal
type gauge struct {
	// field size
	idLen      int
	freqLen    int
	counterLen int

	// indicates one star stands for how many requests or servers
	tableCounterStarUnit     float64
	histogramCounterStarUnit float64
	latencyStarUnit          float64
}

// Visualizer is the visualization module for benchmark
type Visualizer struct {
	clientGroupNames []string
	serverGroupNames []string
	serverData       map[string]*serverGroupMeta
	clientData       map[string]*clientGroupMeta

	gauge *gauge
}

func newServerGroupMeta(groupName string, servers []*Server, buckets []int64) *serverGroupMeta {
	requestCount := int64(0)
	counters := make([]int64, len(servers))
	for i, server := range servers {
		requestCount += server.counter
		counters[i] = server.counter
	}
	histogram := NewHistogram(buckets, 1)
	for _, counter := range counters {
		histogram.IncBucket(counter)
	}

	// all servers have same latency configuration, median of log normal
	// distribution is the mean latency by definition
	meanLatency := servers[0].latency.Median()

	return &serverGroupMeta{
		name:         groupName,
		servers:      servers,
		requestCount: requestCount,
		histogram:    histogram,
		meanLatency:  meanLatency,
	}
}

func newClientGroupMeta(groupName string, clients []*Client) (*clientGroupMeta, error) {
	rps := int(time.Second / clients[0].sleeper.Median())
	reqCount, resCount := int64(0), int64(0)
	histogram := NewHistogram(BucketMs, int64(time.Millisecond))
	for _, client := range clients {
		reqCount += client.reqCounter.Load()
		resCount += client.resCounter.Load()
		if err := histogram.MergeBucket(client.histogram); err != nil {
			return nil, err
		}
	}
	return &clientGroupMeta{
		name:        groupName,
		count:       len(clients),
		reqCount:    reqCount,
		resCount:    resCount,
		histogram:   histogram,
		rps:         rps,
		meanLatency: time.Duration(float64(histogram.WeightedSum())/float64(histogram.Sum())) * time.Millisecond,
	}, nil
}

// aggregate servers into a hash table, also returns maximum request count and
// minimum request at the same time
func aggregateServersByGroupName(servers []*Server) ([]string, map[string][]*Server, int64, int64) {
	serverGroupNames := make([]string, 0)
	serversByGroup := make(map[string][]*Server)
	maxRequestCount, minRequestCount := int64(0), int64(math.MaxInt64)

	for _, server := range servers {
		if server.counter > maxRequestCount {
			maxRequestCount = server.counter
		}
		if server.counter < minRequestCount {
			minRequestCount = server.counter
		}
		groupName := server.groupName
		if _, ok := serversByGroup[groupName]; !ok {
			serverGroupNames = append(serverGroupNames, groupName)
			serversByGroup[groupName] = make([]*Server, 0)
		}
		serversByGroup[groupName] = append(serversByGroup[groupName], server)
	}
	sort.Strings(serverGroupNames)

	return serverGroupNames, serversByGroup, maxRequestCount, minRequestCount
}

func populateServerData(
	serverGroupNames []string,
	serversByGroup map[string][]*Server,
	buckets []int64,
) (map[string]*serverGroupMeta, int, int64) {
	maxServerCount, maxServerHistogramValue := 0, int64(0)
	serverData := make(map[string]*serverGroupMeta)

	for _, groupName := range serverGroupNames {
		meta := newServerGroupMeta(groupName, serversByGroup[groupName], buckets)
		count, histogramValue := len(meta.servers), meta.histogram.Max()
		if count > maxServerCount {
			maxServerCount = count
		}
		if histogramValue > maxServerHistogramValue {
			maxServerHistogramValue = histogramValue
		}
		serverData[groupName] = meta
	}

	return serverData, maxServerCount, maxServerHistogramValue
}

// gaugeCalculation calculated field width and scalar for stars in histograms
// latencyStarUnit will be calculated after getting clientData
func gaugeCalculation(maxServerCount int, maxRequestCount, maxServerHistogramValue int64) *gauge {
	idLen := normalizeLength(getNumLength(int64(maxServerCount)))
	counterLen := normalizeLength(getNumLength(maxRequestCount))
	freqLen := normalizeLength(getNumLength(maxServerHistogramValue))

	tableCounterStarLen := displayWidth - idLen - counterLen - 2*tabWidth
	tableCounterStarUnit := float64(tableCounterStarLen) / float64(maxRequestCount)

	histogramCounterStarLen := displayWidth - counterLen - freqLen - 2*tabWidth
	histogramCounterStarUnit := float64(histogramCounterStarLen) / float64(maxServerHistogramValue)

	return &gauge{
		idLen:                    idLen,
		freqLen:                  freqLen,
		counterLen:               counterLen,
		tableCounterStarUnit:     tableCounterStarUnit,
		histogramCounterStarUnit: histogramCounterStarUnit,
	}
}

func populateClientData(clients []*Client) ([]string, map[string]*clientGroupMeta, int64, error) {
	clientGroupNames := make([]string, 0)
	clientsByGroup := make(map[string][]*Client)
	clientData := make(map[string]*clientGroupMeta)

	for _, client := range clients {
		groupName := client.groupName
		if _, ok := clientsByGroup[groupName]; !ok {
			clientGroupNames = append(clientGroupNames, groupName)
			clientsByGroup[groupName] = make([]*Client, 0)
		}
		clientsByGroup[groupName] = append(clientsByGroup[groupName], client)
	}
	sort.Strings(clientGroupNames)

	maxLatencyFrequency := int64(0)
	for _, groupName := range clientGroupNames {
		meta, err := newClientGroupMeta(groupName, clientsByGroup[groupName])
		if err != nil {
			return nil, nil, 0, err
		}
		histogramMaxFrequency := meta.histogram.Max()
		if meta.histogram.Max() > maxLatencyFrequency {
			maxLatencyFrequency = histogramMaxFrequency
		}
		clientData[groupName] = meta
	}

	return clientGroupNames, clientData, maxLatencyFrequency, nil
}

// NewVisualizer returns a Visualizer for metrics data visualization
func NewVisualizer(ctx *Context) (*Visualizer, error) {
	// aggregate servers, like group by GroupName in sql
	serverGroupNames, serversByGroup, maxRequestCount, minRequestCount := aggregateServersByGroupName(ctx.Servers)
	// calculate request counter buckets based on range
	buckets := NewRequestCounterBuckets(minRequestCount, maxRequestCount, serverBucketCount)
	// populate necessary data for server group visualization
	serverData, maxServerCount, maxServerHistogramValue := populateServerData(serverGroupNames, serversByGroup, buckets)
	// calculate base unit for a star character in terminal
	gauge := gaugeCalculation(maxServerCount, maxRequestCount, maxServerHistogramValue)
	// populate necessary data for client group visualization
	clientGroupNames, clientData, maxLatencyFrequency, err := populateClientData(ctx.Clients)
	if err != nil {
		return nil, err
	}
	// set base unit for latency graph when get maximum latency frequency
	gauge.latencyStarUnit = float64(latencyHeight) / float64(maxLatencyFrequency)

	return &Visualizer{
		clientGroupNames: clientGroupNames,
		clientData:       clientData,
		serverGroupNames: serverGroupNames,
		serverData:       serverData,
		gauge:            gauge,
	}, nil
}

// visualize a server group
func (sgm *serverGroupMeta) visualizeServerGroup(vis *Visualizer, writer io.Writer) {
	gauge := vis.gauge
	servers, histogram := sgm.servers, sgm.histogram
	idLen, freqLen, counterLen := gauge.idLen, gauge.freqLen, gauge.counterLen
	tableStarUnit, histogramStarUnit := gauge.tableCounterStarUnit, gauge.histogramCounterStarUnit
	name, count, latency, total := sgm.name, len(servers), sgm.meanLatency, sgm.requestCount
	separateLine(writer)

	fmt.Fprintf(writer, `request count histogram of server group %q`+"\n", name)
	fmt.Fprintf(writer, "number of servers: %d, latency: %v, total requests received: %d\n", count, latency, total)

	if count <= serverBucketCount {
		// if you only have less than 10 servers in this group, just display them one by one
		fmt.Fprintf(writer, "\n%*s\t%*s\t%s\n", idLen, "id", counterLen, "reqs", "histogram")
		for _, server := range servers {
			counter := server.counter
			stars := truncateStarCount(float64(counter) * tableStarUnit)
			fmt.Fprintf(writer, "%*v\t%*v\t%s\n", idLen, server.id, counterLen, server.counter, strings.Repeat(bar, stars))
		}
	} else {
		// when you have more than hundreds of servers, display them in a histogram
		fmt.Fprintf(writer, "\n%*s\t%*s\t%s\n", counterLen, "reqs", freqLen, "freq", "histogram")
		for i := 0; i < histogram.bucketLen; i++ {
			stars := truncateStarCount(float64(histogram.counters[i].Load()) * histogramStarUnit)
			fmt.Fprintf(writer, "%*v\t%*v\t%s\n",
				counterLen, histogram.buckets[i], freqLen, histogram.counters[i].Load(), strings.Repeat(bar, stars))
		}
	}
	separateLine(writer)
}

// visualize a client group
func (cgm *clientGroupMeta) visualizeClientGroup(vis *Visualizer, writer io.Writer) {
	gauge := vis.gauge
	name := cgm.name
	histogram := cgm.histogram
	count, rps, reqCount, resCount, meanLatency := cgm.count, cgm.rps, cgm.reqCount, cgm.resCount, cgm.meanLatency
	separateLine(writer)

	fmt.Fprintf(writer, `request latency histogram of client group %q`+"\n", name)
	fmt.Fprintf(writer, "number of clients: %d, rps: %d, request issued: %d, response received: %d mean latency: %v\n",
		count, rps, reqCount, resCount, meanLatency)

	fmt.Fprintln(writer)
	pixels := make([][]byte, latencyHeight)
	for i := 0; i < latencyHeight; i++ {
		pixels[i] = make([]byte, latencyWidth)
		pixels[i][0] = '|'
		for j := 1; j < latencyWidth; j++ {
			pixels[i][j] = ' '
		}
	}

	for j := 0; j < histogram.bucketLen; j++ {
		stars := int(float64(histogram.counters[j].Load()) * gauge.latencyStarUnit)
		for i := 0; i < stars; i++ {
			pixels[latencyHeight-1-i][2*(j+1)] = '*'
		}
	}

	for i := 0; i < latencyHeight; i++ {
		for j := 0; j < latencyWidth; j++ {
			fmt.Fprintf(writer, "%c", pixels[i][j])
		}
		fmt.Fprintln(writer)
	}
	fmt.Fprintf(writer, "%s\n", strings.Repeat("-", latencyWidth))

	numLen := 10
	showCount := latencyWidth / numLen
	fmt.Print(" ")
	for i := 1; i <= showCount; i++ {
		fmt.Fprintf(writer, "%*d", numLen, BucketMs[5*i-1])
	}
	fmt.Fprintln(writer)
	separateLine(writer)
}

func getNumLength(num int64) int {
	return len(fmt.Sprint(num))
}

func normalizeLength(l int) int {
	if l < 0 {
		return 0
	}
	return (l + tabWidth - 1) / 4 * 4
}

func separateLine(writer io.Writer) {
	fmt.Fprintf(writer, "\n%s\n", strings.Repeat(separator, displayWidth))
}

func truncateStarCount(count float64) int {
	stars := int(count)
	if stars < 0 {
		return 0
	} else if count > displayWidth {
		return displayWidth
	}
	return stars
}
