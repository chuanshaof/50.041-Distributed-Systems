package main

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

var wg sync.WaitGroup
var cm *CentralManager
var processors []*Processor
var requests int                         // Keeps track of how many requests left to handle
var requestChannels map[int]chan Request // Use a centralized queue like how a proxy/API works

type Page struct {
	id   int
	data int
}

type PageMetaData struct {
	owner   int
	copySet []int
}

type Request struct {
	processorId int
	accessType  string
}

// CentralManager manages the pages
type CentralManager struct {
	mu       sync.Mutex
	metaData map[int]*PageMetaData // Store metadata of pages
}

// Constructor
func NewCentralManager() *CentralManager {
	return &CentralManager{
		metaData: make(map[int]*PageMetaData),
	}
}

// Processor represents a processor in the system
type Processor struct {
	id     int
	cache  map[int]*Page
	access map[int]string // Access is either Read/Write
}

// Constructor
func NewProcessor(id int, cm *CentralManager) *Processor {
	return &Processor{
		id:     id,
		cache:  make(map[int]*Page),
		access: make(map[int]string),
	}
}

// Handle initial requests from the clients
func (cm *CentralManager) handleRequest(pageId int) {
	for {
		select {
		case request := <-requestChannels[pageId]:
			cm.mu.Lock()
			fmt.Printf("\nSTART: Handling processor [%d] request to %s page %d\n", request.processorId, request.accessType, pageId)

			requester := processors[request.processorId]

			// Check for access types
			if cm.metaData[pageId].owner == -1 {
				// Set owner to the processor
				cm.metaData[pageId].owner = request.processorId

				// Grant access to the requester
				messsageChannel := make(chan string)
				page := &Page{
					id:   pageId,
					data: 1,
				}
				go requester.receivePage(request, page, messsageChannel)
				fmt.Println(<-messsageChannel)
			} else if request.processorId == cm.metaData[pageId].owner {
				// For write requests, invalidate copy sets
				if request.accessType == "write" {
					cm.invalidateCopies(pageId)
				}

				// Grant access to the requester
				messsageChannel := make(chan string)
				go requester.receivePage(request, requester.cache[pageId], messsageChannel)
				fmt.Println(<-messsageChannel)
			} else if request.accessType == "read" {
				// Check if it is currently in a "write" access state, change to "read"
				// This will not happen as the owner is the requester, handled above
				if requester.access[pageId] == "write" {
					requester.access[pageId] = "read"
					continue
				}
				// Check if owner is currently in a "write" access state
				if processors[cm.metaData[pageId].owner].access[pageId] == "write" {
					// Block & wait until access is turned to "read"
					// In this case, we simulate and bypass waiting
					processors[cm.metaData[pageId].owner].access[pageId] = "read"
				}

				// Check if already exists in copyset, send page immediately then
				for _, processorId := range cm.metaData[pageId].copySet {
					if processorId == request.processorId {
						// Grant access to the requester
						replyChannel := make(chan string)
						go requester.receivePage(request, requester.cache[pageId], replyChannel)
						fmt.Println(<-replyChannel)
						continue
					}
				}

				cm.readPage(request, pageId)
			} else if request.accessType == "write" {
				cm.writePage(request, pageId)
			}

			// Print the page entire's metadata
			fmt.Printf("Page %d is now owned by [%d] with %s access\n", pageId, cm.metaData[pageId].owner, processors[cm.metaData[pageId].owner].access[pageId])

			for i, page := range cm.metaData {
				fmt.Printf("[MetaData] Page %d: Owner: %d, CopySet: %v\n", i, page.owner, page.copySet)
			}

			// For the purpose of measuring performance
			time.Sleep(1 * time.Millisecond)
			requests = requests - 1
			cm.mu.Unlock()
		}
	}
}

// Faciliate "READ" requests from the client
func (cm *CentralManager) readPage(request Request, pageId int) {
	// Add processor to copySet
	cm.metaData[pageId].copySet = append(cm.metaData[pageId].copySet, request.processorId)
	fmt.Printf("Adding processor [%d] to copy set of page %d\n", request.processorId, pageId)

	// Handle and search for the page owner & make request to readPage
	owner := cm.metaData[pageId].owner
	fmt.Printf("Read forward to processor [%d]\n", owner)
	replyChannel := make(chan string)
	go processors[owner].sendPage(request, pageId, processors[request.processorId], replyChannel)

	// Blocking call to wait for reply
	fmt.Println(<-replyChannel)
}

// Facilitate "WRITE" requests from the client
func (cm *CentralManager) writePage(request Request, pageId int) {
	// Invalidating copy sets
	cm.invalidateCopies(pageId)

	// Original owner of the page
	originalOwner := cm.metaData[pageId].owner

	// Update the owner
	cm.metaData[pageId].owner = request.processorId
	fmt.Printf("Updating owner of page %d to processor [%d]\n", pageId, request.processorId)

	// Trigger for processor to pass data to another processor
	fmt.Printf("Write forward to processor [%d]\n", originalOwner)
	replyChannel := make(chan string)
	go processors[originalOwner].sendPage(request, pageId, processors[request.processorId], replyChannel)

	// Blocking call to wait for reply
	fmt.Println(<-replyChannel)
}

// Invalidation of ALL copies on server copy set
func (cm *CentralManager) invalidateCopies(pageId int) {
	for _, processorId := range cm.metaData[pageId].copySet {
		fmt.Printf("Invalidating copy set of processor [%d]\n", processorId)

		replyChannel := make(chan string)
		go processors[processorId].invalidateCopy(pageId, replyChannel)
		// Blocking call to wait for reply
		fmt.Printf(<-replyChannel)
		cm.metaData[pageId].copySet = remove(cm.metaData[pageId].copySet, processorId)
	}
}

// Removing item from list/array
func remove(slice []int, val int) []int {
	result := []int{}
	for _, v := range slice {
		if v != val {
			result = append(result, v)
		}
	}
	return result
}

// Invalidation of each copy set on the processor's side
func (p *Processor) invalidateCopy(pageId int, replyChannel chan string) {
	// Invalidate copy sets first
	p.cache[pageId] = nil
	p.access[pageId] = ""
	replyChannel <- fmt.Sprintf("Invalidation confirmation from processor [%d]\n", p.id)
}

// Facilitate the sending of `Page` to another processor
func (p *Processor) sendPage(request Request, pageId int, processor *Processor, replyChannel chan string) {
	// Send a copy of the page to the receiver
	fmt.Printf("Processor [%d] sending page %d to processor [%d]\n", p.id, pageId, processor.id)
	processor.receivePage(request, p.cache[pageId], replyChannel)

	// Invalidate own processor's data & access after sending
	if request.accessType == "write" {
		p.cache[pageId] = nil
		p.access[pageId] = ""
	}
}

// Receiving of `Page` from another processor
func (p *Processor) receivePage(request Request, page *Page, replyChannel chan string) {
	p.cache[page.id] = page
	p.access[page.id] = request.accessType
	replyChannel <- fmt.Sprintf("%s confirmation from processor [%d]", strings.Title(request.accessType), p.id)
}

// Randomize read/write requests from the client
func simulateClientRequests(processor *Processor, pageId int) {
	// Randomizer to send read/write requests
	if rand.Intn(2) == 0 {
		// Read request
		requestChannels[pageId] <- Request{
			processorId: processor.id,
			accessType:  "read",
		}
	} else {
		// Write request
		requestChannels[pageId] <- Request{
			processorId: processor.id,
			accessType:  "write",
		}
	}
}

func main() {
	// Creating initialized variables to play scenario
	processorCount := 10 // Number of processors
	pageCount := 1       // Number of pages
	repeatCount := 10    // Number of experiment repetitions
	requestsCount := 100 // Number of requests per repetition

	cm = NewCentralManager()
	requestChannels = make(map[int]chan Request)

	// Creating processors
	// var processors []*Processor
	for i := 0; i < processorCount; i++ {
		processor := NewProcessor(i, cm)
		// cm.processors[i] = processor // Add processor to CM
		processors = append(processors, processor)
	}

	// Go routine to listen to all the channels
	for i := 0; i < pageCount; i++ {
		// Initialize processors
		processors[i].cache[i] = nil

		// Initialize request queue and listen to it
		requestChannels[i] = make(chan Request)
		cm.metaData[i] = &PageMetaData{
			owner:   -1,
			copySet: []int{},
		}
		go cm.handleRequest(i)
	}

	elapsed := time.Duration(0)
	for i := 0; i < repeatCount; i++ {
		// Start timer to see performance
		start := time.Now()

		requests = requestsCount
		// Simulate randomly making requests to CM
		for i := 0; i < requestsCount; i++ {
			// Randomize which page it should read
			randomPage := rand.Intn(pageCount)
			randomProcessor := rand.Intn(processorCount)
			go simulateClientRequests(processors[randomProcessor], randomPage)
		}

		// Check if requests is < 0
		for {
			if requests == 0 {
				break
			}
		}

		// End timer
		elapsed += time.Since(start)
	}
	fmt.Println("=====================================================================================")
	fmt.Printf("Average time taken across %d repetitions: %s\n", repeatCount, elapsed/time.Duration(repeatCount))
	time.Sleep(1 * time.Second)
}
