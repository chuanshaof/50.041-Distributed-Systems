package main

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

var wg sync.WaitGroup
var cms []*CentralManager
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
	id       int
	mu       sync.Mutex
	metaData map[int]*PageMetaData // Store metadata of pages
	backup   bool
	dead     bool
}

// Constructor
func NewCentralManager(id int) *CentralManager {
	return &CentralManager{
		id:       id,
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
func NewProcessor(id int) *Processor {
	return &Processor{
		id:     id,
		cache:  make(map[int]*Page),
		access: make(map[int]string),
	}
}

// Run constant healthCheck to see if other CM is dead
func (cm *CentralManager) healthCheck(otherCM *CentralManager) {
	for {
		if cm.dead || !cm.backup {
			continue
		}

		// Check if the other CM is dead or a back up now, if so, take over
		if otherCM.dead || otherCM.backup {
			cm.backup = false
			fmt.Println("\n================================================================")
			fmt.Printf("CM [%d] has taken over\n", cm.id)
			fmt.Println("================================================================")
		}
	}
}

// Replication of metaData, always called synchronously from ther other CM
func (cm *CentralManager) replicateMetaData(metaData map[int]*PageMetaData, replicationChannel chan string) {
	if cm.dead {
		replicationChannel <- fmt.Sprintf("CM [%d] is dead, cannot replicate\n", cm.id)
		return
	}

	// Replicate data to other CM
	cm.metaData = metaData
	replicationChannel <- fmt.Sprintf("MetaData replication confirmation from CM [%d]\n", cm.id)
}

// Handle initial requests from the clients
func (cm *CentralManager) handleRequest(pageId int) {
	for {
		// Skip the listener if it is a backup or dead
		if cm.backup || cm.dead {
			continue
		}

		select {
		case request := <-requestChannels[pageId]:
			// fmt.Println(cm.id, cm.dead)
			cm.mu.Lock()
			fmt.Printf("\nSTART: CM [%d] Handling processor [%d] request to %s page %d\n", cm.id, request.processorId, request.accessType, pageId)

			requester := processors[request.processorId]

			// Check for access types
			if cm.metaData[pageId].owner == -1 {
				// Set owner to the processor
				cm.metaData[pageId].owner = request.processorId

				// Replication of metaData to ensure consistency
				replicationChannel := make(chan string)
				go cms[cm.id^1].replicateMetaData(cm.metaData, replicationChannel)
				fmt.Printf(<-replicationChannel)

				// Grant access to the requester
				replyChannel := make(chan string)
				page := &Page{
					id:   pageId,
					data: 1,
				}
				go requester.receivePage(request, page, replyChannel)
				fmt.Println(<-replyChannel)
			} else if request.processorId == cm.metaData[pageId].owner {
				// For write requests, invalidate copy sets
				if request.accessType == "write" {
					cm.invalidateCopies(pageId)
				}

				// Grant access to the requester
				replyChannel := make(chan string)
				go requester.receivePage(request, requester.cache[pageId], replyChannel)
				fmt.Println(<-replyChannel)
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

					// Replication of metaData to ensure consistency
					replicationChannel := make(chan string)
					go cms[cm.id^1].replicateMetaData(cm.metaData, replicationChannel)
					fmt.Printf(<-replicationChannel)
				}

				cm.readPage(request, pageId)
			} else if request.accessType == "write" {
				cm.writePage(request, pageId)
			}

			// Print the page entire's metadata
			fmt.Printf("Page %d is now owned by [%d] with %s access\n", pageId, cm.metaData[pageId].owner, processors[cm.metaData[pageId].owner].access[pageId])

			for i, page := range cm.metaData {
				fmt.Printf("[CM %d MetaData] Page %d: Owner: %d, CopySet: %v\n", cm.id, i, page.owner, page.copySet)
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

	// Replication of metaData to ensure consistency
	replicationChannel := make(chan string)
	go cms[cm.id^1].replicateMetaData(cm.metaData, replicationChannel)
	fmt.Printf(<-replicationChannel)

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

	// Replication of metaData to ensure consistency
	replicationChannel := make(chan string)
	go cms[cm.id^1].replicateMetaData(cm.metaData, replicationChannel)
	fmt.Printf(<-replicationChannel)

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

		// Replication of metaData to ensure consistency
		replicationChannel := make(chan string)
		go cms[cm.id^1].replicateMetaData(cm.metaData, replicationChannel)
		fmt.Printf(<-replicationChannel)
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
		// fmt.Printf("Processor [%d] request to %s page %d\n", processor.id, "read", pageId)
	} else {
		// Write request
		requestChannels[pageId] <- Request{
			processorId: processor.id,
			accessType:  "write",
		}
		// fmt.Printf("Processor [%d] request to %s page %d\n", processor.id, "write", pageId)
	}
}

func main() {
	// Initialization of CMs
	mainCM := NewCentralManager(0)
	backupCM := NewCentralManager(1)
	backupCM.backup = true
	cms = append(cms, mainCM)
	cms = append(cms, backupCM)

	// Initialize request queue and listen to it
	requestChannels = make(map[int]chan Request)

	// Start healthCheck between the other
	go cms[0].healthCheck(cms[1])
	go cms[1].healthCheck(cms[0])

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
		requestChannels[i] = make(chan Request)

		// Initialize request queue and listen to it
		cms[0].metaData[i] = &PageMetaData{
			owner:   -1,
			copySet: []int{},
		}

		cms[1].metaData[i] = &PageMetaData{
			owner:   -1,
			copySet: []int{},
		}
		go cms[0].handleRequest(i)
		go cms[1].handleRequest(i)
	}

	// Start timer to see performance
	start := time.Now()

	requests = 100
	requestsCount := requests
	// Simulate randomly making requests to CM
	for i := 0; i < requestsCount; i++ {
		// Randomize which page it should read
		randomPage := rand.Intn(pageCount)
		randomProcessor := rand.Intn(processorCount)
		go simulateClientRequests(processors[randomProcessor], randomPage)
	}

	// Simulate one random failure
	// time.Sleep(1 * time.Millisecond)
	// go func() {
	// 	fmt.Println("\n================================================================")
	// 	fmt.Println("Simulating failure of main CM")
	// 	fmt.Println("================================================================")
	// 	mainCM.dead = true
	// }()

	// Check if requests is < 0
	for {
		if requests == 0 {
			break
		}
	}

	// End timer
	elapsed := time.Since(start)

	fmt.Println("=====================================================================================")
	fmt.Printf("Time taken: %s\n", elapsed)
	time.Sleep(1 * time.Second)
}
