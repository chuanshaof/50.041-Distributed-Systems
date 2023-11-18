package main

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

var wg sync.WaitGroup

type MessageType int

const (
	Request MessageType = iota
	Vote
	Release
	Rescind
)

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
	if q.Len() == 0 {
		return
	} else {
		*q = (*q)[1:]
	}
}

// Node struct
type Node struct {
	ID            int
	Clock         int
	VoteFor       int          // -1 if not voted, otherwise ID of the node voted for
	Queue         RequestQueue // Queue to manage incoming requests
	VotesReceived map[int]bool // Votes received from other nodes
	Mutex         sync.Mutex
	MsgChannel    chan Message
	Peers         []*Node
}

func (n *Node) RequestAccess() {
	defer wg.Done()

	// Wait until whoever it has voted for has released the lock
	n.Mutex.Lock()
	if n.VoteFor != -1 {
		// Rescind the previous vote
		rescindMsg := Message{Type: Rescind, Timestamp: n.Clock, SenderID: n.ID}
		for _, peer := range n.Peers {
			if peer.ID == n.VoteFor {
				peer.MsgChannel <- rescindMsg
			}
		}
	}
	n.Mutex.Unlock()

	// time.Sleep(time.Duration(rand.Intn(1)) * time.Second)

	n.Mutex.Lock()
	n.Clock++
	n.VoteFor = n.ID
	msg := Message{Type: Request, Timestamp: n.Clock, SenderID: n.ID}

	for _, peer := range n.Peers {
		peer.MsgChannel <- msg
		n.VotesReceived[peer.ID] = false
	}
	n.Mutex.Unlock()

	for {
		votes := 1
		for _, v := range n.VotesReceived {
			if v {
				votes++
			}
		}

		// Adjusted for the fact that the node votes for itself
		if votes > (len(n.Peers)+1)/2 {
			break // out of `while` loop
		}
		time.Sleep(10 * time.Millisecond)
	}

	n.Mutex.Lock()
	// fmt.Printf("Node %d entering critical section\n", n.ID)
	// time.Sleep(time.Duration(rand.Intn(1)+1) * time.Second)
	// fmt.Printf("Node %d leaving critical section\n", n.ID)

	time.Sleep(100 * time.Millisecond)

	releaseMsg := Message{Type: Release, Timestamp: n.Clock, SenderID: n.ID}
	for _, peer := range n.Peers {
		peer.MsgChannel <- releaseMsg
	}
	n.VoteFor = -1

	// Upon release, vote for the next node as well
	if n.Queue.Len() > 0 {
		nextReq := n.Queue[0]
		voteMsg := Message{Type: Vote, Timestamp: n.Clock, SenderID: n.ID}
		n.VoteFor = nextReq.SenderID
		for _, peer := range n.Peers {
			if peer.ID == nextReq.SenderID {
				peer.MsgChannel <- voteMsg
			}
		}
	}

	n.Mutex.Unlock()
}

func (n *Node) Listen() {
	for msg := range n.MsgChannel {
		n.Mutex.Lock()
		n.Clock = max(n.Clock+1, msg.Timestamp)
		switch msg.Type {
		case Request:
			n.Queue.Add(msg) // Add incoming requests to the queue

			// Yet to have voted for anyone, send out request
			if n.VoteFor == -1 {
				firstReq := n.Queue[0]
				voteMsg := Message{Type: Vote, Timestamp: n.Clock, SenderID: n.ID}
				n.VoteFor = firstReq.SenderID
				for _, peer := range n.Peers {
					if peer.ID == firstReq.SenderID {
						peer.MsgChannel <- voteMsg
					}
				}
			} else {
				if n.Queue[0].SenderID != n.VoteFor {
					// Send a rescind vote to the previous voter if necessary
					rescindMsg := Message{Type: Rescind, Timestamp: n.Clock, SenderID: n.ID}
					for _, peer := range n.Peers {
						if peer.ID == n.VoteFor {
							peer.MsgChannel <- rescindMsg
						}
					}
				}
			}

		case Release:
			// Release vote
			if n.VoteFor != n.ID {
				n.VoteFor = -1
			}

			// Process the release message
			// For the case where it is not a rescinded one
			if n.Queue.Len() > 0 && n.Queue[0].SenderID == msg.SenderID {
				n.Queue.Pop() // Remove the request of the sender from the queue
			}

			// NOTE: We do not need to pop and add back because the queue is already sorted

			// Vote for the next request in the queue
			// Could be both for rescinded & actual releases
			if n.VoteFor != n.ID {
				if n.Queue.Len() > 0 {
					nextReq := n.Queue[0]
					voteMsg := Message{Type: Vote, Timestamp: n.Clock, SenderID: n.ID}
					n.VoteFor = nextReq.SenderID
					for _, peer := range n.Peers {
						if peer.ID == nextReq.SenderID {
							peer.MsgChannel <- voteMsg
						}
					}
				}
			}

		case Vote:
			// For receiving a vote
			n.VotesReceived[msg.SenderID] = true

		case Rescind:
			// For receiving request to rescind
			n.VotesReceived[msg.SenderID] = false
			releaseMsg := Message{Type: Release, Timestamp: n.Clock, SenderID: n.ID}

			// Send release message back
			for _, peer := range n.Peers {
				if peer.ID == msg.SenderID {
					peer.MsgChannel <- releaseMsg
				}
			}
		}
		n.Mutex.Unlock()
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
		ID:            id,
		MsgChannel:    make(chan Message, 10),
		VoteFor:       -1,
		VotesReceived: make(map[int]bool),
	}
	return node
}

func main() {
	count := 10

	// Setting up of nodes, requires locking
	mu := sync.Mutex{}
	mu.Lock()
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
	}

	// Start each node's message handling in a separate goroutine
	for _, node := range nodes {
		go node.Listen()
	}
	mu.Unlock()

	for i := 1; i <= 10; i++ {
		startTime := time.Now()
		// Run concurrently if necessary
		for j := 1; j <= i; j++ {
			wg.Add(1)
			go func(n *Node) {
				n.RequestAccess()
			}(nodes[j-1])
		}

		wg.Wait()
		duration := time.Since(startTime)
		fmt.Println("Time taken for", i, "concurrent nodes:", duration)
	}

	// for _, node := range nodes {
	// 	wg.Add(1)
	// 	go func(n *Node) {
	// 		n.RequestAccess()
	// 	}(node)
	// }

	wg.Wait()
}
