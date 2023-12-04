# Overview
This Go pset simulates a distributed cache system, incorporating concepts like page management, request handling, and failure resilience in a multi-processor environment. It showcases a system where processors make read/write requests to a central manager (CM) which controls the page access. The system is designed to handle replication, request processing, and failure simulations to mimic real-world scenarios in distributed systems.

This implementation is able to handle multiple Pages, Processors, and Requests. However, this will only handle 2 CMs (main and backup) as according to the assignment requirements. 

Leader election is simplified, where CM[0] will always be the leader and will take over if it comes back alive. The backup CM will only take over if the main CM is down. 

There will be no individual queues so as to simplify the problem as well. A global variable called `requestChannels` has been introduced to handle requests. This is similar to a reverse proxy or a load balancer for the both CM, where there is a central API for the clients to interact with.

Types of messages passed:
- Read/Write request (P to CM)
- Read/Write confirmation (P to CM)
- Invalidate copy confirmation (P to CM)
- Page transfer/send (P to P)
- Read/Write forward (CM to P)
- Invalidate copy (CM to P)
- Data replication (CM to CM)

## How to Run:
Follow these steps to run any file in this project. Replace `<folder>` and `<file.go>` as needed:

1. Ensure Go is installed on your machine.
2. Navigate to the directory containing the program.
3. Run the Go file using the following commands:

```bash
cd pset3/<folder>
go run <file.go>
```

## External Packages:
No external packages are required for this project.

## Additional Flags:
No additional flags are needed as the simulation runs in a pre-defined sequence. Please follow the command-line output for a step-by-step understanding of the simulation process.

<br><br><br>

# ivy.go

## Overview
This Go package implements a simulated distributed cache system. It focuses on the interactions between processors and central managers (CMs), handling data page requests and managing read/write operations in a distributed environment. 

## Struct Information
### CentralManager
- **MetaData**: A map storing metadata for each page.

### Processor
- **ID**: Unique identifier for the processor.
- **Cache**: Local cache storage for pages.
- **Access**: Map tracking access type (read/write) for each page.

### Page
- **ID**: Page identifier.
- **Data**: Actual data contained in the page.

### Request
- **ProcessorID**: ID of the requesting processor.
- **AccessType**: Type of access requested (read/write).

## CentralManager Methods
### HandleRequest
A method in `CentralManager` that listens for and processes page requests from processors. It handles granting access, coordinating read/write operations, and replicating metadata.

### ReadPage
A method in `CentralManager` that handles read requests from processors. It checks if the page is in the cache, and if not, it coordinates the read operation with the processor and other CMs.

### WritePage
A method in `CentralManager` that handles write requests from processors. It checks if the page is in the cache, and if not, it coordinates the write operation with the processor and other CMs.

### InvalidateCopies
A method in `CentralManager` that invalidates copies of a page in its copyset.

## Processor Methods
### InvalidateCopy
A method in `Processor` that invalidates a copy of a page in its cache.

### SendPage
A method in `Processor` that sends a page to another processor. 

### ReceivePage
A method in `Processor` that receives a page from another processor. Includes sending a confirmation message to the CM.

## Simulation Methods
### SimulateClientRequests
A function that generates random read/write requests from processors to simulate real-world usage.

### Main
The `main` function initializes the CMs and processors, setting up the simulation environment. 

## Performance Metrics
The output includes time taken for request processing under various conditions, such as different numbers of processors and pages.

## Interpretation of Results
- The simulation output shows how requests are handled.
- Time metrics indicate the system's efficiency in processing requests and recovering from failures.
- Observations can be made on how the system scales with an increasing number of processors.

<br><br><br>

# ivyBackup.go

## Overview
This package builds upon `ivy.go`. In this readme, I will only be covering the additional features on top of `ivy.go`.

This package includes a main CM and a backup CM for resilience, and it showcases how requests are processed and how the system behaves under various failure scenarios.

## Struct Information
### CentralManager
- **ID**: Unique identifier for the CM.
- **Backup**: Indicates if the CM is a backup.
- **Dead**: Status flag indicating if the CM is operational or has failed.

## CentralManager Methods
### HealthCheck
A method in `CentralManager` that continuously checks the status of the other CM, taking over if the main CM fails.

### ReplicateMetaData & ReplicateReply
Methods in `CentralManager` that replicates metadata to the backup CM with message reply waiting. These methods are called whenever there is a change in metadata. In the case where the other CM is down, there will be a timeout to simulate waiting and failure to replicate data. This will affect performance.

## Simulation Methods
### GoesDown & ComesBack
These methods help to simulate a CM going down. It will flip the CM's `dead` flag to determine if it should continue with `HandleRequest` routine. It will not occur when `inAccess` is true, which is not a fault scenario that is handled in this implementation.

## Simulation Cases & Settings
From the assignment, there are multiple simulations that have been designed to measure the performances. To run these simulations, simply change the `simulationCase` variable in `main` to the desired case number. The various scenarios are:
0. Normal operation
1. CM[0] goes down and comes up once
2. CM[0] goes down and comes up multiple times at intervals of 150 milliseconds
3. CM[0] or CM[1] goes down and comes up multiple times at intervals of 150 milliseconds

# Summary of simulation results
The below table shows an average time taken (across 10 repetitions) to complete these simulations. They are ran with the following settings:
- 10 processors
- 1 page
- 100 requests

| Simulation Case | Time Taken (s) | 
| --------------- | --------------- |
| No Fault Tolerant | 1.451 |
| 0 | 1.455 | 
| 1 | 1.532 |
| 2 | 2.278 | 
| 3 | 2.628 |

Refer to the PDF within this folder for a detailed explanation of the results.


