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
This is a Go program that simulates a distributed mutual-exclusion algorithm mimicking Lamport's Priority Queue. 


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


# lamport.go
This is a Go program that simulates a server-client message-passing system. Clients send messages to the server and the server can either forward these messages to all other clients or drop them. 

The server has a 50% chance of either forwarding or dropping an incoming message. The program simulates 10 clients sending 2 messages each, and can be adjusted in the code through the variable `numOfMessages`. 

Message ordering at the end is determined by the Lamport Clock, which is a logical clock to order events in a distributed system.


## Interpreting the Output:
The output will display logs of:

- Starting the protocol.
- Clients sending messages to the server.
- The server forwarding or dropping the messages.
- All processed messages sorted by their Lamport Clock.

## Go Routine Comments:
1. `SendMessage`: Sends a message from a client to the server with a certain delay.
2. `PollForMessage` (Client): Listens for incoming messages to a client and adjusts the client's Lamport Clock accordingly.
3. `PollForMessage` (Server): Listens for incoming messages to the server and either forwards them to all clients or drops them.
4. `RegisterClient`: Registers a client with the server.
5. `main`: Initiates the server, registers clients, and starts the protocol. After processing everything, it orders and prints messages by their Lamport Clock.

<br><br><br>

# vector.go
This is a Go program that simulates a server-client message-passing system. Clients send messages to the server and the server can either forward these messages to all other clients or drop them. 

The server has a 50% chance of either forwarding or dropping an incoming message. The program simulates 10 clients sending 2 messages each, and can be adjusted in the code through the variable `numOfMessages`. 

Messages that violate causality are stored and displayed at the end of the simulation.

## Interpreting the Output:

1. Protocol initialization message.
2. Logs displaying clients sending messages to the server.
3. The server either forwarding or dropping messages.
4. Detected causality violations based on vector clocks, with related message details.

## Go Routine Comments:

1. `SendMessage`: A routine where clients send messages to the server. 
2. `PollForMessage` (Client): Listens for incoming messages and adjusts the client's vector clock. It also checks for causality violations.
3. `PollForMessage` (Server): Listens for incoming messages and forwards or drops them based on a random choice.
4. `RegisterClient`: Registers a client with the server.
5. `main`: Initiates the server, registers clients, and starts the protocol. Once all messages are processed, it prints all causality violations.

<br><br><br>

# bully.go

This code provides a basic implementation and simulation of the **Bully Algorithm** for leader election in distributed systems. 

In the Bully Algorithm, when a process identifies that the coordinator (or leader) is no longer operational, it initiates an election process. The process with the highest ID (or significance) typically wins the election.

## Main Components

- **Message Struct**: Messages used for communication among nodes. Contains:
  - Sender's ID
  - Message content, including:
    - "ELECTION" for initiating an election
    - "COORDINATOR" for declaring leadership
    - "ACCEPT" and "DECLINE" for election responses.

- **Node Struct**: Represents each node in the system. Properties include:
  - Node ID
  - Alive status
  - Coordinator details
  - Election and data channels
  - Election flags

- **synchronize() Method**: Manages synchronization among nodes. If a node doesn't receive data within a specified timeframe, it considers the coordinator dead and starts an election.

- **startElection() Method**: A node begins the election process. It sends election requests to nodes with higher IDs. If none respond within a given timeframe, it elects itself as the coordinator. Alternatively, if the node receives a rejection, it fails the election.

- **listenForElections() Method**: Nodes continuously listen for incoming election messages and respond accordingly. 

- **goesDown() Method**: Simulates a node crash.

- **refreshNodes() Function**: Resets all nodes to initial stages being alive and the highest node being the coordinator.

## Main Execution Flow

- Initialization of nodes and starting essential goroutines such as `synchronize` and `listenForElections`
- The Bully algorithm is then simulated through several scenarios:
  1. A basic initial election.
  2. Best-case and worst-case node failure detections.
  3. Scenarios of election interruptions.
  4. Simultaneous election starts.
  5. Silent node exits.