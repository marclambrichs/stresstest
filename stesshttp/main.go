package main

import (
	"flag"
	"fmt"
	"github.com/mlambrichs/stresstest/alphabet"
	"github.com/mlambrichs/stresstest/alphabet/file"
	"github.com/mlambrichs/stresstest/alphabet/nato"
	"gopkg.in/fatih/pool.v2"
	"log"
    "math/rand"
	"net"
	"runtime"
	"strconv"
	"sync"
    "time"
)

var (
	nrOfApplications    int
	path                string
	poolCapacity        int
	port                int
	server              string
	timeout             int

	waitGrp             sync.WaitGroup
)

func init() {
	const (
		defaultNrOfApplications = 20
		defaultPoolCapacity     = 20
		defaultPort             = 80
		defaultServer           = "127.0.0.1"
		defaultTimeout          = 60

		usageNrOfApplications   = "the number of metrics being sent"
		usagePath               = "the path of your inputfile containing alphabet"
		usagePoolCapacity       = "the size of the pool of connections"
		usagePort               = "the port to connect to"
		usageServer             = "the server to connect to"
		usageTimeout            = "timeout between sent messages of same metric"
	)
	// define flag for nr of applications
	flag.IntVar(&nrOfApplications, "nr_of_applications", defaultNrOfApplications, usageNrOfApplications)
	flag.IntVar(&nrOfApplications, "n", defaultNrOfApplications, usageNrOfApplications+" (shorthand)")

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

// check if application doesn't already exist
func isNotNewApplication(application string, applications map[string]int) bool {
	_, ok := applications[application]
	return ok
}

func main() {
	var alphabet alphabet.Alphabet

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
    // Pool has minimum capacity of 5.
	p, err := pool.NewChannelPool(5, poolCapacity, factory)
	if err != nil {
		log.Fatal(err)
	}

	// create a map for containing al applications
	applications := make(map[string]int)

	// select your alphabet as a base for your applications
	switch path != "" {
	case true:
		alphabet = file.NewBuffer(path)
	default:
		var a nato.Nato
		alphabet = a
	}

	log.Printf("Starting with %d applications", nrOfApplications)
	for i := 0; i < nrOfApplications; i++ {
		// create new application
		var newApplication string
		for {
			newApplication, _ = alphabet.Get()
			if !isNotNewApplication(newApplication, applications) {
				break
			}
		}
		// add 1 to waitGroup
		waitGrp.Add(1)
		// open up a channel for this specific metric
		c := make(chan string)
		// ...and start sending GETs right away
		go Send(newApplication, timeout, c)
		// kick off the receiver
		go receiver(p, c)
		// ...and save new metric into map
		applications[newApplication]++
		log.Printf("new application %s", newApplication)
	}

	waitGrp.Wait()
	// Close pool. This means closing all connedctions in pool.
	p.Close()
}

func Send(application string, timeout int, c chan string) {
    start := rand.Intn(59)
	time.Sleep(time.Duration(start) * time.Second)
	for {
        c <- fmt.Sprintf("GET /%s HTTP/1.0\r\n\r\n", application)
		time.Sleep(time.Duration(timeout) * time.Second)
	}
}

func receiver(p pool.Pool, c chan string) {
	var (
		conn net.Conn
		err  error
	)
	for {
		msg := <-c
		log.Printf("application %s received", msg)
		conn, err = p.Get()
		if err != nil {
			break
		}
		fmt.Fprintf(conn, msg+"\n")
		conn.Close()
	}
	waitGrp.Done()
	log.Fatalf("application %s stopped", err)
}
