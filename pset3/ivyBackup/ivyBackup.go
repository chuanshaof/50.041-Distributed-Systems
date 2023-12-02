package main

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

var wg sync.WaitGroup
var mainCM *CentralManager
var backupCM *CentralManager
var processors []*Processor
var requests int

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
	id             int
	mu             sync.Mutex
	metaData       map[int]*PageMetaData // Store metadata of pages
	requestChannel map[int]chan Request  // To ensure sequential access to the page
	backup         bool
	dead           bool
}

// Constructor
func NewCentralManager(id int) *CentralManager {
	return &CentralManager{
		id:             id,
		metaData:       make(map[int]*PageMetaData),
		requestChannel: make(map[int]chan Request),
	}
}

// Processor represents a processor in the system
type Processor struct {
	id     int
	cache  map[int]*Page
	access map[int]string // Access is either Read/Write
}

// Constructor
func NewProcessor(id int) *Processor {
	return &Processor{
		id:     id,
		cache:  make(map[int]*Page),
		access: make(map[int]string),
	}
}

func (cm *CentralManager) healthCheck(otherCM *CentralManager) {
	for {
		if cm.dead || !cm.backup {
			continue
		}

		// Check if the other CM is dead or a back up now, if so, take over
		if otherCM.dead || otherCM.backup {
			cm.backup = false
			fmt.Printf("Central Manager [%d] is now the main CM\n", cm.id)
		}
	}
}

// Replication of metaData
func (cm *CentralManager) replicateMetaData(metaData map[int]*PageMetaData, replyChannel chan string) {
	// Replicate data to other CM
	cm.metaData = metaData
	replyChannel <- fmt.Sprintf("MetaData replication confirmation from CM [%d]\n", cm.id)
}

// Replication of requests
func (cm *CentralManager) replicateRequest(request Request, pageId int, replyChannel chan string) {
	// Send a copy of the page to the receiver
	cm.requestChannel[pageId] <- request
	replyChannel <- fmt.Sprintf("Request replication confirmation from CM [%d]\n", cm.id)
}

// Handle initial requests from the clients
func (cm *CentralManager) handleRequest(pageId int) {
	for {
		// Skip the listener if it is a backup or dead
		if cm.backup || cm.dead {
			continue
		}

		select {
		case request := <-cm.requestChannel[pageId]:
			cm.mu.Lock()

			fmt.Printf("START: Handling processor [%d] request to %s page %d\n", request.processorId, request.accessType, pageId)

			// For main server, check if CM is dead and if there is a need to replicate
			// There is no need to check for backup server, see README for explaination
			if cm != backupCM && !backupCM.dead {
				// Replicate request to backup
				// FIXME: The replication has issues, sometimes before the replication is done, the main CM is dead
				fmt.Printf("Replicating request to CM [%d]\n", backupCM.id)
				replyChannel := make(chan string)
				go backupCM.replicateRequest(request, pageId, replyChannel)
				fmt.Printf(<-replyChannel)
			}

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

				cm.readPage(request, pageId)
			} else if request.accessType == "write" {
				cm.writePage(request, pageId)
			}

			// Print the page entire's metadata
			fmt.Printf("Page %d is now owned by [%d] with %s access\n", pageId, cm.metaData[pageId].owner, processors[cm.metaData[pageId].owner].access[pageId])

			for i, page := range cm.metaData {
				fmt.Printf("[MetaData] Page %d: Owner: %d, CopySet: %v\n", i, page.owner, page.copySet)
			}
			fmt.Printf("\n")

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

	// Trigger for processor to pass data to another processor
	owner := cm.metaData[pageId].owner
	fmt.Printf("Write forward to processor [%d]\n", owner)
	replyChannel := make(chan string)
	go processors[owner].sendPage(request, pageId, processors[request.processorId], replyChannel)

	// Blocking call to wait for reply
	fmt.Println(<-replyChannel)

	cm.metaData[pageId].owner = request.processorId
}

// Invalidation of ALL copies on server copy set
func (cm *CentralManager) invalidateCopies(pageId int) {
	for _, processorId := range cm.metaData[pageId].copySet {
		fmt.Printf("Invalidating copy set of processor [%d]\n", processorId)

		replyChannel := make(chan string)
		go processors[processorId].invalidateCopy(pageId, replyChannel)
		// Blocking call to wait for reply
		fmt.Printf(<-replyChannel)
	}
	cm.metaData[pageId].copySet = []int{}
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
		mainCM.requestChannel[pageId] <- Request{
			processorId: processor.id,
			accessType:  "read",
		}
		// fmt.Printf("Processor [%d] request to %s page %d\n", processor.id, "read", pageId)
	} else {
		// Write request
		mainCM.requestChannel[pageId] <- Request{
			processorId: processor.id,
			accessType:  "write",
		}
		// fmt.Printf("Processor [%d] request to %s page %d\n", processor.id, "write", pageId)
	}
}

func main() {
	mainCM = NewCentralManager(0)
	backupCM = NewCentralManager(1)
	backupCM.backup = true

	go mainCM.healthCheck(backupCM)
	go backupCM.healthCheck(mainCM)

	// Creating processors
	processorCount := 10
	// var processors []*Processor
	for i := 0; i < processorCount; i++ {
		processor := NewProcessor(i)
		// cm.processors[i] = processor // Add processor to CM
		processors = append(processors, processor)
	}

	pageCount := 1
	// Go routine to listen to all the page channels
	for i := 0; i < pageCount; i++ {
		// Initialize processors
		processors[i].cache[i] = nil

		// Initialize request queue and listen to it
		mainCM.requestChannel[i] = make(chan Request, 10)
		mainCM.metaData[i] = &PageMetaData{
			owner:   -1,
			copySet: []int{},
		}

		backupCM.requestChannel[i] = make(chan Request, 10)
		backupCM.metaData[i] = &PageMetaData{
			owner:   -1,
			copySet: []int{},
		}
		go mainCM.handleRequest(i)
		go backupCM.handleRequest(i)
	}

	// Start timer to see performance
	start := time.Now()

	requests = 10
	requestsCount := requests
	// Simulate randomly making requests to CM
	for i := 0; i < requestsCount; i++ {
		// Randomize which page it should read
		randomPage := rand.Intn(pageCount)
		randomProcessor := rand.Intn(processorCount)
		go simulateClientRequests(processors[randomProcessor], randomPage)
	}

	// Simulate one random failure
	time.Sleep(1 * time.Millisecond)
	go func() {
		mainCM.dead = true
	}()

	// Check if requests is < 0
	for {
		if requests == 0 {
			break
		}
	}

	// End timer
	elapsed := time.Since(start)
	fmt.Printf("Time taken: %s\n", elapsed)
	time.Sleep(1 * time.Second)
}
