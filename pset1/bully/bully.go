package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const nodeCount = 5

var timeOut = time.Second * 5
var syncTime = time.Second * 2

var nodes [nodeCount]*Node
var mutex sync.RWMutex
var wg sync.WaitGroup

/*
content must be an enum of "ELECTION", "COORDINATOR", "ACCEPT", "DECLINE"
"ELECTION" means elect itself
"COORDINATOR" means announce that it is the coordinator
"ACCEPT" means to accept request to elect
"DECLINE" means to decline request to elect
*/
type Message struct {
	sender  int
	content string
}

// NOTE: Assume that data is always updated by the coordinator
type Node struct {
	// Keep track of who is coord, etc
	id            int
	isAlive       bool
	coordinator   int
	isCoordinator bool

	// Data sync purposes
	dataChannel chan string
	data        string

	// Election purposes
	electionChannel chan Message
	declined        bool
	electing        bool
}

func (node *Node) synchronize() {
	for {
		if node.isCoordinator && node.isAlive {
			for _, n := range nodes {
				if n.id != node.id && n.isAlive {
					n.dataChannel <- node.data
				}
			}
		} else if node.isAlive {
			node.data = <-node.dataChannel

			// Check if channel is empty and if to detect if coordinator is down
			if len(<-node.dataChannel) == 0 {
				time.Sleep(timeOut * 2)
				if len(<-node.dataChannel) == 0 {
					wg.Add(1)
					go node.startElection(-1)
					wg.Wait()
				}
			}
		}
	}
}

func (node *Node) startElection(failMidWay int) {
	defer wg.Done()

	node.electing = true
	node.declined = false
	fmt.Printf("Node %d started an election.\n", node.id)
	time.Sleep(time.Second * 1)

	for _, n := range nodes {
		if n.id > node.id {
			n.electionChannel <- Message{node.id, "ELECTION"}
		}
	}

	// Start time
	startTime := time.Now()

	// Polling loop
	for {
		// Check if `timeOut` seconds have passed
		if time.Since(startTime) > timeOut {
			break
		}

		if node.declined {
			fmt.Println("Node", node.id, "has failed the election.")
			return
		}
	}

	// Comes here if node is not declined
	if !node.declined {
		node.electing = false
		node.isCoordinator = true
		for _, n := range nodes {
			if n.isAlive && node.isAlive {

				// For 3a.
				if n.id == failMidWay {
					n.isAlive = false
					fmt.Println("Node", n.id, "has died while broadcasting.")
					continue
				}

				if n.id != node.id {
					n.electionChannel <- Message{node.id, "COORDINATOR"}
				}
			}
		}

		if !(node.id == failMidWay) {
			fmt.Printf("Node %d is now the coordinator.\n", node.id)
		}
	} else {
		fmt.Println("Node", node.id, "has failed the election.")
	}
}

func (node *Node) listenForElections() {
	for {
		select {
		case m := <-node.electionChannel:
			if !node.isAlive {
				continue
			}

			if m.content == "ELECTION" {
				// Check if node id is higher than sender
				if node.id > m.sender {
					nodes[m.sender].electionChannel <- Message{node.id, "DECLINE"}

					if node.electing == false {
						wg.Add(1)
						go node.startElection(-1)
					}
				} else {
					nodes[m.sender].electionChannel <- Message{node.id, "ACCEPT"}
				}
			} else if m.content == "COORDINATOR" {
				node.coordinator = m.sender
				continue
			} else if m.content == "DECLINE" {
				node.declined = true
			} else if m.content == "ACCEPT" {
				continue
			}
		}
	}
}

func (node *Node) goesDown() {
	node.isAlive = false
	fmt.Printf("Node %d goes down.\n", node.id)
}

func main() {
	for i := range nodes {
		node := &Node{
			id:            i,
			isAlive:       true,
			isCoordinator: false,
			coordinator:   -1,

			dataChannel: make(chan string, 10),
			data:        "",

			electionChannel: make(chan Message, 10),
			declined:        true,
			electing:        false,
		}
		nodes[i] = node
		go node.listenForElections()
		go node.synchronize()
	}

	// 1. Test if election and joint synchronization works
	fmt.Println("\nStarting initial election...")
	wg.Add(1)
	go nodes[nodeCount-1].startElection(-1)
	wg.Wait()
	refreshNodes()

	// 2. Simulate scenarios where 1 node goes down
	// Best Case Scenario: node #n-1 detects failure
	fmt.Println("\nStarting best-case scenario...")
	nodes[nodeCount-1].goesDown()
	wg.Add(1)
	go nodes[nodeCount-2].startElection(-1)
	wg.Wait()
	refreshNodes()

	// Worst Case Scenario: node 0 detects failure
	fmt.Println("\nStarting worst-case scenario...")
	wg.Add(1)
	nodes[nodeCount-1].goesDown()
	go nodes[0].startElection(-1)
	wg.Wait()
	refreshNodes()

	// 3.
	// 	a. Elected coordinator fails while announcing
	fmt.Println("\nStarting election failure scenario a...")
	wg.Add(1)
	nodes[nodeCount-1].goesDown()
	go nodes[nodeCount-2].startElection(nodeCount - 2)
	wg.Wait()
	refreshNodes()

	// 	b. The failed node is not the newly elected coordinator
	fmt.Println("\nStarting election failure scenario b...")
	wg.Add(1)
	nodes[nodeCount-1].goesDown()
	go nodes[nodeCount-2].startElection(0)
	wg.Wait()
	refreshNodes()

	// 4. Multiple GO routines start the election process simultaneously
	// 		Probably just start multiple goroutines for handleElection, maybe 2 & 3
	fmt.Println("\nStarting simultaneous election scenario...")
	wg.Add(1)
	nodes[nodeCount-1].goesDown()
	go nodes[0].startElection(-1)
	wg.Wait()
	refreshNodes()

	// 5. Node silently leaves
	// Choose a random node using math/rand
	fmt.Println("\nStarting silent leave scenario...")
	randNode := rand.Intn(nodeCount)
	randNode = 4
	nodes[randNode].goesDown()
	fmt.Println("Node", randNode, "silently leaves.")

	// If it is coordinator, wait to see if anyone elects itself?
	if nodes[randNode].isCoordinator {
		time.Sleep(3 * timeOut)
	}
}

func refreshNodes() {
	mutex.Lock()
	for i := range nodes {
		node := &Node{
			id:            i,
			isAlive:       true,
			isCoordinator: false,
			coordinator:   nodeCount - 1,

			dataChannel: make(chan string, 10),
			data:        "",

			electionChannel: make(chan Message, 10),
			declined:        true,
			electing:        false,
		}
		nodes[i] = node
		go node.listenForElections()
		go node.synchronize()
	}

	nodes[nodeCount-1].isCoordinator = true
	mutex.Unlock()
}
