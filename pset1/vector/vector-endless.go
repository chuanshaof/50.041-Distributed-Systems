package main

// import (
// 	"fmt"
// 	"math/rand"
// 	"strconv"
// 	"sync"
// 	"time"
// )

// var wg sync.WaitGroup
// var numOfClients = 10
// var messageId = 0

// /* Message struct */
// type Message struct {
// 	Id          int
// 	VectorClock []int
// }

// /* Client struct */
// type Client struct {
// 	Id          int
// 	VectorClock []int
// 	Outgoing    chan Message
// 	Incoming    chan Message
// }

// func (c *Client) SendMessage() {
// 	defer wg.Done()
// 	for {
// 		// Send a message very 10-20 seconds
// 		time.Sleep(time.Duration(rand.Intn(10)+10) * time.Second)

// 		// Add 1 to own clock when sending
// 		c.VectorClock[c.Id]++

// 		messageId++
// 		m := Message{
// 			Id:          messageId,
// 			VectorClock: c.VectorClock,
// 		}

// 		fmt.Println("Message ID: " + strconv.Itoa(m.Id) + " Client " + strconv.Itoa(c.Id) + " sending to server.")
// 		c.Outgoing <- m
// 	}
// }

// func (c *Client) ReceiveMessage() {
// 	for {
// 		select {
// 		case m := <-c.Incoming:
// 			// Handle Message VectorClock
// 			c.HandleVectorClock(m)

// 			fmt.Println("Message ID: " + strconv.Itoa(m.Id) + " Client " + strconv.Itoa(c.Id) + " receiving.")
// 		}
// 	}
// }

// func (c *Client) HandleVectorClock(m Message) {
// 	// Update and detect if theres causality violation

// 	// Handle the case to merge both vector clocks
// 	for i := 0; i < numOfClients; i++ {
// 		// In the case of i == c.Id, there will never be a causality fault
// 		if i == c.Id {
// 			if m.VectorClock[i] > c.VectorClock[i] {
// 				c.VectorClock[i] = m.VectorClock[i] + 1
// 			} else {
// 				c.VectorClock[i]++
// 			}
// 		} else if m.VectorClock[i] >= c.VectorClock[i] {
// 			c.VectorClock[i] = m.VectorClock[i]
// 		} else if m.VectorClock[i] < c.VectorClock[i] {
// 			// Here is when causality violation happens, when the incoming vector clock is less
// 			fmt.Println("Causality violation detected in Message ID: " + strconv.Itoa(m.Id) + " Client " + strconv.Itoa(c.Id) + " receiving.")
// 		}
// 	}
// }

// /* Server struct */
// type Server struct {
// 	Clients []Client
// }

// func (s *Server) RegisterClient(c *Client) {
// 	s.Clients = append(s.Clients, *c)
// }

// func (s *Server) PollForMessage() {
// 	for {
// 		for i, c := range s.Clients {
// 			select {
// 			case m := <-c.Outgoing:
// 				// Forward message to all other clients
// 				s.ForwardMessage(&s.Clients[i], m)
// 			}
// 		}
// 	}
// }

// func (s *Server) ForwardMessage(c *Client, m Message) {
// 	if rand.Intn(2) == 0 {
// 		fmt.Println("Message ID: " + strconv.Itoa(m.Id) + " Server dropping message from client " + strconv.Itoa(c.Id))
// 		return
// 	}

// 	for i := 0; i < numOfClients; i++ {
// 		if i != c.Id {
// 			fmt.Println("Message ID: " + strconv.Itoa(m.Id) + " Server forwarding message to client " + strconv.Itoa(i))

// 			s.Clients[i].Incoming <- m
// 		}
// 	}
// }

// func main() {
// 	fmt.Println("Running")

// 	s := Server{}

// 	go s.PollForMessage()

// 	for i := 0; i < numOfClients; i++ {
// 		outgoing := make(chan Message)
// 		incoming := make(chan Message)
// 		c := &Client{
// 			Id:          i,
// 			VectorClock: make([]int, numOfClients),
// 			Outgoing:    outgoing,
// 			Incoming:    incoming,
// 		}
// 		s.RegisterClient(c)

// 		wg.Add(2)
// 		go c.SendMessage()
// 		go c.ReceiveMessage()
// 	}

// 	wg.Wait()
// }
