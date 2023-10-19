package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"time"
)

var wg sync.WaitGroup
var s Server
var numOfClients = 10
var numOfMessages = 2
var messageId = 0
var allMessages []Message

/* Message struct */
type Message struct {
	Id           int
	Sender       int
	Receiver     int
	LamportClock int
}

/* Client struct */
type Client struct {
	Id           int
	LamportClock int
	Incoming     chan Message
}

func (c *Client) SendMessage() {
	for i := 0; i < numOfMessages; i++ {
		// Send a message very X seconds
		time.Sleep(time.Duration(rand.Intn(1)+1) * time.Second)

		c.LamportClock++
		messageId++

		m := Message{
			Id:           messageId,
			LamportClock: c.LamportClock,
			Sender:       c.Id,
		}

		fmt.Println("Message ID: " + strconv.Itoa(m.Id) + " Client " + strconv.Itoa(c.Id) + " sending to server.")
		s.Incoming <- m
	}
}

func (c *Client) PollForMessage() {
	for {
		select {
		case m := <-c.Incoming:
			if m.Id != 0 {
				// Handle Message LamportClock, take max(local, incoming) + 1
				if m.LamportClock > c.LamportClock {
					c.LamportClock = m.LamportClock
				}
				c.LamportClock++
				m.Receiver = c.Id

				allMessages = append(allMessages, m)
				fmt.Println("Message ID: " + strconv.Itoa(m.Id) + " Client " + strconv.Itoa(c.Id) + " receiving. LamportClock: " + strconv.Itoa(c.LamportClock))
			}
		}
	}
}

/* Server struct */
type Server struct {
	Clients  []Client
	Incoming chan Message
}

func (s *Server) RegisterClient(c *Client) {
	s.Clients = append(s.Clients, *c)
}

func (s *Server) PollForMessage() {
	for {
		select {
		case m := <-s.Incoming:
			if m.Id != 0 {
				// Forward message to all other clients
				// 1 to forward message, otherwise drops
				if rand.Intn(2) == 1 {
					for i := 0; i < numOfClients; i++ {
						if i != m.Sender {
							fmt.Println("Message ID: " + strconv.Itoa(m.Id) + " Server forwarding message to client " + strconv.Itoa(i))
							s.Clients[i].Incoming <- m
						}
					}
				} else {
					fmt.Println("Message ID: " + strconv.Itoa(m.Id) + " Server dropping message from client " + strconv.Itoa(m.Sender))
				}
			}
		}
	}
}

func main() {
	fmt.Println("Starting protocol...")

	s = Server{
		Incoming: make(chan Message, numOfClients*numOfMessages),
	}

	go s.PollForMessage()

	for i := 0; i < numOfClients; i++ {
		c := &Client{
			Id:           i,
			LamportClock: 0,
			Incoming:     make(chan Message, numOfClients*numOfMessages),
		}
		s.RegisterClient(c)

		go c.PollForMessage()
		go c.SendMessage()
	}

	// Wait for messages to be processsed
	time.Sleep(10 * time.Second)

	wg.Add(1)
	closeAndEmpty(s.Incoming)
	for _, c := range s.Clients {
		wg.Add(1)
		closeAndEmpty(c.Incoming)
	}
	wg.Wait()

	fmt.Println("Starting ordering...")
	// sort all messages by lamport clock
	sort.Slice(allMessages, func(i, j int) bool {
		return allMessages[i].LamportClock < allMessages[j].LamportClock
	})

	for _, m := range allMessages {
		fmt.Println("Message ID: " + strconv.Itoa(m.Id) + " Sender: " + strconv.Itoa(m.Sender) + " Receiver: " + strconv.Itoa(m.Receiver) + " LamportClock: " + strconv.Itoa(m.LamportClock))
	}

	time.Sleep(10 * time.Second)
}

func closeAndEmpty(ch chan Message) {
	defer wg.Done()
	close(ch)
	for len(ch) > 0 {
		<-ch
	}
}
