# TASKS:
- DONE: read from config file or environment variable
(OR dynamically read config from own state written by another program (database reader etc.))
- Configurable parameters:
	- DONE: port
	- DONE: websocket route(s)
	- DONE: websocket buffer sizes
	- DONE: stream channel size
	- later: database credentials (if authentication is handled by this server)
	- authentication: USER, TOKEN, JWT
	- authorization: ACL, RBAC, etc.
- DONE: save current state of resources before terminating (or even periodically)
- DONE: load last state of resources at startup
- DONE: LINK operation to link a resource to another resource

# TODO:
- Limit upload size! DONE (in websocket)
- Creation and deletion of resources: preloaded from config/auth, dynamically created/deleted, swap in/out
- Authentication (microservice or fully integrated?)
- Savepointing overhaul (one file per resource or one big file like now?)
	- reimplement with new directory
- Monitoring (using prometheus / influx?)
- finish TCP and solve length header problem
	- streaming deserialization with msgp

other resource impl:
O(1) efficient bounded dynamically allocated but partially pre-allocated thread safe queue with multiple read-ends for low-latency applications in golang
-> Kafka / Haskell Chan