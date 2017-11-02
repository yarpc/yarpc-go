// Package shortestqueue provides an implementation of a peer list that sends
// traffic to the peer with the fewest pending requests, but degenerates to
// round robin when all peers are equally loaded.
package shortestqueue
