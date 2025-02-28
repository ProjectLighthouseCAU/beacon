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
- DONE: replace all occurences of "interface{}" with "any"
- DONE: change STREAM behavior: allow multiple streams of same resource but with different REID
- DONE: decide if all streams of a resource get the updates in the same order (racing put requests cannot be ordered) -> ANSWER: no, for now

- DECIDE: if values should be dropped if stream is slow (and not slow down other streams) -> ANSWER: yes, for most payloads (e.g. images), maybe later add config to not drop values in resource

- IMPORTANT/DIFFICULT: websocket timeout after some time of no or only invalid, or unauthorized request

- IMPORTANT/MEDIUM-DIFFICULT: overhaul interface to heimdall (redis seems unergonomic and clunky), maybe use http endpoint in heimdall to check authentication and cache the result
- UNIMPORTANT/MEDIUM: overhaul snapshotting (make it more robust, maybe change "gob" AND "msgpack" to just "msgpack" -> cannot differentiate between directory and resource content)
- UNIMPORTANT/MEDIUM: export prometheus metrics
- UNIMPORTANT/DIFFICULT: fine grained access control

# TODO:

- Limit upload size! DONE (in websocket)
- Creation and deletion of resources: preloaded from config/auth, dynamically created/deleted, swap in/out
- Authentication (microservice or fully integrated?)
- Savepointing overhaul (one file per resource or one big file like now?)
  - reimplement with new directory
- Monitoring (using prometheus / influx?)
- finish TCP and solve length header problem
  - streaming deserialization with msgp

## other resource impl:

- O(1) efficient bounded dynamically allocated but partially pre-allocated thread safe queue with multiple read-ends for low-latency applications in golang
- -> Kafka / Haskell Chan

- dynamically allocated linked list is very slow, go channels have a fixed size and one continuous underlying array of memory (ring buffer)
- implement a ring buffer using a fixed size array
- keep the write end synchronized using a lock
- keep track of the number of readers
- remove a value from the ring buffer only when all readers have read it (this could be difficult)

### try using channels first:

- currently using one channel as input, a goroutine as a broker/forwarder, a variable with mutex and n channels for n streams
- we would like to remove the goroutine as it consumes memory for every resource
- Resource: Variable with Mutex + n channels (directly without broker goroutine)
- Put: lock write mutex -> write variable -> send to all channels -> unlock write mutex
  - alternatively unlock before sending to the channels
  - thread safe, but:
    - streams might get the values in different orders
  - only do this if it really improves performance
- Get: lock read mutex -> read variable -> unlock read mutex -> return value
- Stream: lock write mutex -> add another stream to list/map of streams -> unlock write mutex
- Stop: similar to stream but remove
- Link: Same as Stream but:
  - set "stream" channel of source to the input channel of the destination
  - no need for a goroutine to forward the values
