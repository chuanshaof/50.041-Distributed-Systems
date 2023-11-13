package main

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"
)

var wg sync.WaitGroup

type MessageType int

const (
	Request MessageType = iota
	Acknowledge
	Release
)

// Message struct, `Type` is an enum
type Message struct {
	Type      MessageType
	Timestamp int
	SenderID  int
}

/*
Provides interface for sort.Sort

Sorting functionality is based on timestamp and senderID

	Lower timestamp first
	Lower senderID next
*/
type RequestQueue []Message

func (q RequestQueue) Len() int {
	return len(q)
}

func (q RequestQueue) Less(i, j int) bool {
	if q[i].Timestamp == q[j].Timestamp {
		return q[i].SenderID < q[j].SenderID
	}
	return q[i].Timestamp < q[j].Timestamp
}

func (q RequestQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}

// Add a message to the queue and sort
func (q *RequestQueue) Add(msg Message) {
	*q = append(*q, msg)
	sort.Sort(q)
}

func (q *RequestQueue) Pop() {
	// Remove first item in the queue
	*q = (*q)[1:]
}

// Node struct
type Node struct {
	ID           int
	Clock        int
	Queue        RequestQueue
	Acknowledges map[int]bool
	Mutex        sync.Mutex
	MsgChannel   chan Message
	Peers        []*Node
	RequestTime  int
}

// RequestAccess simulates a node requesting access to a critical section
func (n *Node) RequestAccess() {
	defer wg.Done()
	time.Sleep(time.Duration(rand.Intn(1)) * time.Second) // simulate random requests

	// Requesting access to critical section
	// Increment logical clock by 1 and add request to queue
	n.Clock++
	n.RequestTime = n.Clock
	msg := Message{Type: Request, Timestamp: n.Clock, SenderID: n.ID}
	n.Queue.Add(msg)

	// Send request message to all peers & set all acknowledgments to false
	n.broadcastRequest(msg)

	n.Mutex.Lock()
	for _, peer := range n.Peers {
		n.Acknowledges[peer.ID] = false
	}
	n.Mutex.Unlock()

	// Wait for acknowledgments from all peers
	for {
		n.Mutex.Lock()
		allAcked := true
		for _, acked := range n.Acknowledges {
			if !acked {
				allAcked = false
				break // out of `for`` loop
			}
		}

		if allAcked {
			break // out of `while`` loop
		}
		n.Mutex.Unlock()
		time.Sleep(10 * time.Millisecond) // simulate waiting & polling
	}

	// Critical section
	fmt.Printf("Node %d entering critical section\n", n.ID)
	time.Sleep(time.Duration(rand.Intn(1)+1) * time.Second) // simulate critical section work
	fmt.Printf("Node %d leaving critical section\n", n.ID)

	// Removing itself from its own queue once work is done

	n.Queue.Pop()

	// Send release message to all peers
	n.broadcastRelease()
	n.RequestTime = -1

	// Acknowledge other requests in the queue
	for _, msg := range n.Queue {
		ackMsg := Message{Type: Acknowledge, Timestamp: n.Clock, SenderID: n.ID}
		for _, peer := range n.Peers {
			if peer.ID == msg.SenderID {
				peer.MsgChannel <- ackMsg
			}
		}
	}
}

// Broadcasts a request message to all peers
func (n *Node) broadcastRequest(msg Message) {
	for _, peer := range n.Peers {
		peer.MsgChannel <- msg
	}
}

// Broadcasts a release message to all peers
func (n *Node) broadcastRelease() {
	releaseMsg := Message{Type: Release, Timestamp: n.Clock, SenderID: n.ID}
	for _, peer := range n.Peers {
		peer.MsgChannel <- releaseMsg
	}
}

// Listen to all messages coming in on the node's message channel
func (n *Node) Listen() {
	for msg := range n.MsgChannel {
		switch msg.Type {
		case Request:
			// fmt.Println("Node", n.ID, "received message of request from node", msg.SenderID, "with timestamp", msg.Timestamp)

			// Update node logical clock and add request to queue
			n.Clock = max(n.Clock+1, msg.Timestamp)

			if n.RequestTime < msg.Timestamp || (n.RequestTime == msg.Timestamp && n.ID < msg.SenderID) {
				n.Queue.Add(msg)
			}

			// Reply to requests if it has no requests or if its request is behind
			if n.RequestTime == -1 || (n.RequestTime > msg.Timestamp || (n.RequestTime == msg.Timestamp && n.ID > msg.SenderID)) {
				ackMsg := Message{Type: Acknowledge, Timestamp: n.Clock, SenderID: n.ID}
				for _, peer := range n.Peers {
					if peer.ID == msg.SenderID {
						peer.MsgChannel <- ackMsg
					}
				}
			}

		// Poll for release requests
		case Release:
			// fmt.Println("Node", n.ID, "received message of release from node", msg.SenderID, "with timestamp", msg.Timestamp)

			// Update logical clock and remove request from queue
			n.Clock = max(n.Clock+1, msg.Timestamp)

		// Waiting for nodes to acknowledge its own request
		case Acknowledge:
			// fmt.Println("Node", n.ID, "received message of acknowledge from node", msg.SenderID, "with timestamp", msg.Timestamp)

			n.Mutex.Lock()
			n.Acknowledges[msg.SenderID] = true
			n.Mutex.Unlock()
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Creates a new node with the given ID and peers
func NewNode(id int) *Node {
	node := &Node{
		ID:          id,
		MsgChannel:  make(chan Message, 10),
		RequestTime: -1,
	}
	node.Acknowledges = make(map[int]bool)
	return node
}

func main() {
	count := 8
	// Create a network of nodes
	var nodes []*Node
	for i := 0; i < count; i++ {
		nodes = append(nodes, NewNode(i))
	}

	// Set each node's peers, making each other known
	for i := range nodes {
		for j := range nodes {
			if i != j {
				nodes[i].Peers = append(nodes[i].Peers, nodes[j])
			}
		}
		// nodes[i].Peers = nodes
	}

	// Start each node's message handling in a separate goroutine
	for _, node := range nodes {
		go node.Listen()
	}

	// Simulate each node requesting access individually
	for _, node := range nodes {
		wg.Add(1)
		go func(n *Node) {
			n.RequestAccess()
		}(node)
	}

	wg.Wait()

	// Wait for a while to allow all nodes to process
	// time.Sleep(100 * time.Second)
}
