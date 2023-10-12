package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

var wg sync.WaitGroup
var numOfClients = 10

type Client struct {
	Id      int
	Message chan string
}

func (c *Client) SendMessage() {
	// Send a message very 1-10 seconds
	time.Sleep(time.Duration(rand.Intn(10)) * time.Second)

	c.Message <- strconv.Itoa(c.Id) + " Sending message"
}

type Server struct {
	Clients []Client
}

func (s *Server) ListenMessage() {
	for _, c := range s.Clients {
		select {
		case m := <-c.Message:
			// Go through all the other channels just to figure out what to pass back?
			fmt.Println(m)
		}
	}
}

func (s *Server) RegisterClient(c *Client) {
	s.Clients = append(s.Clients, *c)
}

func (s *Server) ForwardMesage(c *Client) {
	if rand.Intn(2) == 0 {
		return
	}

	for i := 0; i < numOfClients; i++ {

	}

	// select {
	// case m := <-message:
	// 	fmt.Println(m)
	// }
}

func main() {
	fmt.Println("Running")

	s := Server{}

	for i := 0; i < numOfClients; i++ {
		channel := make(chan string)
		c := &Client{
			Id:      i,
			Message: channel,
		}
		s.RegisterClient(c)

		wg.Add(1)
		go c.SendMessage()
	}

	go s.ListenMessage()

	for {
		select {}
	}

	// time.Sleep(100000 * time.Second)
}

// func main() {
// 	    arr := []uint{3, 4, 5, 6, 1, 2}

// 	    c := make(chan []uint)

// 	    var wg sync.WaitGroup

// 	    go func(c chan []uint, wg *sync.WaitGroup) {
// 	        wg.Add(1)
// 	        defer wg.Done()
// 	        nArr := <-c
// 	        fmt.Println(nArr)
// 	        nArr[1] = 0
// 	    }(c, &wg)

// 	    c <- arr
//
// 	    time.Sleep(time.Second)
// 	    fmt.Println(arr)
// 	    wg.Wait()
// 	}
