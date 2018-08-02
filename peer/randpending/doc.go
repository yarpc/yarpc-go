// package randpending provides a load balancer implementation that
// picks two peers at random and chooses the one with fewer pending requests.
//
// The Power of Two Choices in Randomized Load Balancing:
// https://www.eecs.harvard.edu/~michaelm/postscripts/tpds2001.pdf
//
// The Power of Two Random Choices: A Survey of Techniques and Results:
// https://www.eecs.harvard.edu/~michaelm/postscripts/handbook2001.pdf
package randpending
