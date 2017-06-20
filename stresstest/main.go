package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/mlambrichs/stresstest/alphabet"
	"github.com/mlambrichs/stresstest/alphabet/file"
	"github.com/mlambrichs/stresstest/alphabet/nato"
	"github.com/mlambrichs/stresstest/metric"
	"gopkg.in/fatih/pool.v2"
	"log"
	"math/big"
	"net"
	"runtime"
	"strconv"
	"sync"
)

var (
	nrOfMetrics  int
	path         string
	poolCapacity int
	port         int
	server       string
	timeout      int

	waitGrp sync.WaitGroup
)

func init() {
	const (
		defaultNrOfMetrics  = 20000
		defaultPoolCapacity = 20
		defaultPort         = 2003
		defaultServer       = "127.0.0.1"
		defaultTimeout      = 60

		usageNrOfMetrics  = "the number of metrics being sent"
		usagePath         = "the path of your inputfile containing alphabet"
		usagePoolCapacity = "the size of the pool of connections"
		usagePort         = "the port to connect to"
		usageServer       = "the server to connect to"
		usageTimeout      = "timeout between sent messages of same metric"
	)
	// define flag for nr of metrics
	flag.IntVar(&nrOfMetrics, "nr_of_metrics", defaultNrOfMetrics, usageNrOfMetrics)
	flag.IntVar(&nrOfMetrics, "n", defaultNrOfMetrics, usageNrOfMetrics+" (shorthand)")

	// define flag for path
	flag.StringVar(&path, "path", "", usagePath)
	flag.StringVar(&path, "p", "", usagePath+" (shorthand)")

	// define flag for pool capacity
	flag.IntVar(&poolCapacity, "pool_capacity", defaultPoolCapacity, usagePoolCapacity)
	flag.IntVar(&poolCapacity, "pc", defaultPoolCapacity, usagePoolCapacity+" (shorthand)")

	// define flag for port
	flag.IntVar(&port, "port", defaultPort, usagePort)
	flag.IntVar(&port, "po", defaultPort, usagePort+" (shorthand)")

	// define flag for server
	flag.StringVar(&server, "server", defaultServer, usageServer)
	flag.StringVar(&server, "s", defaultServer, usageServer+"(shorthand)")

	flag.IntVar(&timeout, "timeout", defaultTimeout, usageTimeout)
	flag.IntVar(&timeout, "t", defaultTimeout, usageTimeout)
}

// check if meetric doesn't already exist
func isNotNewMetric(metric metric.Metric, metrics map[string]int) bool {
	_, ok := metrics[metric.String()]
	return ok
}

// calculate depth
func calculateDepth(nrOfMetrics int, alphabethLength int) (k int, err error) {
	log.Printf("calculateDepth(%d, %d)", nrOfMetrics, alphabethLength)
	n := big.NewInt(int64(nrOfMetrics))
	a := int64(alphabethLength)
	k = 1
	result := new(big.Int)

	for {
		if int64(k) > a || big.NewInt(int64(k)).Cmp(n) == 0 {
			err = errors.New("Oops. Numbers aren't big enough.")
			break
		} else if result.Binomial(a, int64(k)); result.Cmp(n) > 0 {
			break
		}
		log.Printf("binomial(%d, %d) =  %s", a, k, result.Binomial(a, int64(k)))
		k++
	}
	return
}

func main() {
	var (
		alphabet alphabet.Alphabet
		depth    int
	)

	// parse commandline flags
	flag.Parse()

	numcpu := runtime.NumCPU()
	// GOMAXPROCS limits the number of operating system threads that can
	// execute user-level Go code simultaneously
	runtime.GOMAXPROCS(numcpu)

	// create factory function to be used with channel based pool
	connection := net.JoinHostPort(server, strconv.Itoa(port))
	factory := func() (net.Conn, error) { return net.Dial("tcp", connection) }

	// create a channel based pool to manage all connections
	p, err := pool.NewChannelPool(5, poolCapacity, factory)
	if err != nil {
		log.Fatal(err)
	}

	// create a map for containing al metrics
	metrics := make(map[string]int)

	// select your alphabet
	switch path != "" {
	case true:
		alphabet = file.NewBuffer(path)
	default:
		var a nato.Nato
		alphabet = a
	}

	// calculate the number of parts a metric should exist of
	depth, err = calculateDepth(nrOfMetrics, alphabet.Len())
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Starting with %d metrics and depth %d", nrOfMetrics, depth)
	for i := 0; i < nrOfMetrics; i++ {
		// create new metric
		var newMetric metric.Metric
		for {
			newMetric = metric.New(alphabet, depth)
			if !isNotNewMetric(newMetric, metrics) {
				break
			}
		}
		// add 1 to waitGroup
		waitGrp.Add(1)
		// open up a channel for this specific metric
		c := make(chan string)
		// ...and start generating metrics right away
		go newMetric.Send(timeout, c)
		// kick off the receiver
		go receiver(p, c)
		// ...and save new metric into map
		metrics[newMetric.String()]++
		log.Printf("new metric %s", newMetric.String())
	}

	waitGrp.Wait()
	// Close pool. This means closing all connedctions in pool.
	p.Close()
}

func receiver(p pool.Pool, c chan string) {
	var (
		conn net.Conn
		err  error
	)
	for {
		msg := <-c
		log.Printf("metric %s received", msg)
		conn, err = p.Get()
		if err != nil {
			break
		}
		fmt.Fprintf(conn, msg+"\n")
		conn.Close()
	}
	waitGrp.Done()
	log.Fatalf("metric %s stopped", err)
}
