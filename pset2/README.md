# Overview
Each file will have its own README sections and the documentation on how to interpret the outcome as shown below.

All problems have been solved utilizing Scalar Clock + machine ID as stipulated by the correction on edimension.

## How to Run:
The same method to run is used for all files. Please replace \<folder\> and \<file.go\> as accordingly.

1. Ensure you have Go installed on your machine.
2. Navigate to the directory where the program is located.
3. Use `go run` to run the file. There will be no command line arguments required.
```
cd pset2/<folder>
go run <file.go>
```

## External Packages:
No external packages were used for this entire pset.

## Additional flags required:
No additional flags will be required in this entire pset as all simulation has been planned out in sequence. Please read the output in the command line to understand the scenario at each step.

<br><br><br>

# lamportQueue.go

## Overview

This Go package provides an implementation of Lamport Priority Queue algorithm. It simulates a distributed system of nodes communicating and coordinating access to a critical section. It uses a message passing mechanism with logical clocks for synchronization. Each node in the system can request access to a critical section, receive acknowledgments from its peers, and send a release message upon exiting the critical section. This system helps to understand concepts like distributed consensus, race conditions, and the coordination of parallel processes in a distributed environment.

## Struct Information
### Message
- **Type**: The type of the message, represented by an enum (`Request`, `Acknowledge`, `Release`).
- **Timestamp**: A logical timestamp used for ordering messages.
- **SenderID**: The ID of the node sending the message.

### RequestQueue
A slice of `Message` that provides methods for queue operations such as add and pop, with custom sorting logic based on timestamps and sender IDs.

### Node
- **ID**: A unique identifier for the node.
- **Clock**: A logical clock to maintain the order of events.
- **Queue**: A queue of messages representing requests to access the critical section.
- **Acknowledges**: A map tracking acknowledgments from other nodes.
- **Mutex**: A mutex to ensure atomicity of operations.
- **MsgChannel**: A channel for receiving messages from other nodes.
- **Peers**: A list of pointers to peer nodes.
- **RequestTime**: The time at which the node made a request to access the critical section.

## Go Routine Information
### RequestAccess
A method of `Node` that simulates a node requesting access to a critical section. It sends a request to all peers and waits for acknowledgments. Once all acknowledgments are received, the node proceeds to the critical section.

### Listen
A method of `Node` that listens for incoming messages on the node's message channel. Based on the message type (`Request`, `Acknowledge`, `Release`), it performs actions like updating the logical clock, adding requests to the queue, and acknowledging requests.

### main
The `main` function initializes a network of nodes and starts each node's message handling in a separate goroutine. It simulates the nodes requesting access to the critical section concurrently and measures the time taken for the operations.

Time taken for 1 concurrent nodes: 320.576ms
Time taken for 2 concurrent nodes: 641.0551ms
Time taken for 3 concurrent nodes: 944.9917ms
Time taken for 4 concurrent nodes: 1.2891332s
Time taken for 5 concurrent nodes: 1.5990818s
Time taken for 6 concurrent nodes: 1.9301639s
Time taken for 7 concurrent nodes: 2.2434907s
Time taken for 8 concurrent nodes: 2.6373256s
Time taken for 9 concurrent nodes: 2.9612435s
Time taken for 10 concurrent nodes: 3.2255014s

# ricartAgrawala.go
## Overview
This Go package is an advanced implementation of the Ricart and Agrawala's algorithm in distributed systems where nodes communicate and coordinate access to a critical section. It involves message passing with logical clocks for synchronization and handles concurrent requests and acknowledgments in a distributed environment. The primary focus is on managing access to a shared resource among multiple nodes, ensuring correct sequence and fair access.


## How is it different from `lamportQueue.go`?
Rather than keeping track of all requests and having to reply to every node, Ricart and Agrawala only keeps track of the requests that are of a lower priority to itself. 

As a result, there are no release request, reducing the number of calls it needs to make.


Time taken for 1 concurrent nodes: 322.2789ms
Time taken for 2 concurrent nodes: 634.3518ms
Time taken for 3 concurrent nodes: 964.1507ms
Time taken for 4 concurrent nodes: 1.3061782s
Time taken for 5 concurrent nodes: 1.6235079s
Time taken for 6 concurrent nodes: 1.8809189s
Time taken for 7 concurrent nodes: 2.2747769s
Time taken for 8 concurrent nodes: 2.571568s
Time taken for 9 concurrent nodes: 2.884402s
Time taken for 10 concurrent nodes: 3.1669846s

# votingProtocol.go
# README for Voting-Based Distributed Node Communication and Access Control in Go

## Overview

This Go package presents the implementation of a Voting Protocol distributed system where nodes utilize a voting mechanism to manage access to a critical section. It's designed to demonstrate a more complex coordination strategy in a distributed environment, where nodes cast votes to decide which one gains access to a shared resource. This system helps in understanding distributed consensus, voting algorithms, and handling concurrent requests in a network of nodes.

In this section, I will only highlight the differences from the above 2 protcols.

## Struct Information
### Node

- **ID**: Unique identifier for the node.
- **Clock**: Logical clock to maintain order of events.
- **VoteFor**: ID of the node this node is voting for (-1 if not voted).
- **Queue**: A `RequestQueue` holding incoming pending requests.
- **VotesReceived**: Map tracking votes received from other nodes.
- **Mutex**: Mutex for synchronization.
- **MsgChannel**: Channel for receiving messages from peers.
- **Peers**: List of pointers to other nodes (peers) in the system.

## Go Routine Information

### RequestAccess
Handles a node's request for accessing a critical section. It involves:
- Rescinding previous votes, if any.
- Incrementing the node's logical clock.
- Voting for itself and requesting votes from peers.
- Waiting for a majority of votes before entering the critical section.
- On exiting, releasing the vote and voting for the next request in the queue, if any.

### Listen
Listens for incoming messages and processes them based on their type. Actions include:
- Adding requests to the queue.
- Voting for requests and managing rescind and release messages.
- Updating the logical clock and handling vote acknowledgments.
- Releasing votes and changing votes based on the state of the queue.


Time taken for 1 concurrent nodes: 318.7813ms
Time taken for 2 concurrent nodes: 653.07ms
Time taken for 3 concurrent nodes: 988.6204ms
Time taken for 4 concurrent nodes: 1.322439s
Time taken for 5 concurrent nodes: 1.5798892s
Time taken for 6 concurrent nodes: Impossible
Time taken for 7 concurrent nodes: Impossible
Time taken for 8 concurrent nodes: Impossible
Time taken for 9 concurrent nodes: Impossible
Time taken for 10 concurrent nodes: Impossible