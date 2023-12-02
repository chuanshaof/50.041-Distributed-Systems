This implementation assumes that there can be multiple pages which are centrally managed by a central manager

Furthermore, the processors themselves are able to handle and hold multiple pages.

A global variable has been introduced so as to remove the simulation requirement using RPC calls or channels. It still works as according to what is required in the assignment.

To ensure that Ivy is sequentially consistent, I introduced a requestChannel that acts a queue for individual pages such that requests are read in a sequential manner.

Types of messages passed:
- Read/Write request (P to CM)
- Read/Write forward (CM to P)
- Invalidate copy (CM to P)
- Page transfer/send (P to P)
- Read/Write confirmation (P to CM)
- Invalidate copy confirmation (P to CM)

To make things simple for this case, as there are only 2 servers (main and backup), the main server will be the one that takes control. The backup server will only take control when the main server is down. Therefore, the main server will be the only one that requires to replicate data.

The backup server will never need to replicate data as it will only take control when the main server is down, making there no need to replicate data.