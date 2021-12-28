# The Lighthouse Server (Beacon)
### **B**asis for **E**very **A**pplication with **C**onnections **O**ver **N**etworks

## About
At Project Lighthouse we wanted to provide our users (mostly students) with a platform for building animations and games to be played on our big screen, the "Lighthouse" - an LED installation inside a 14 story university building.  
Because users need to test their applications against our API, we needed a kind of virtual Lighthouse for that.  
Such a thing would also make management during events/shows much easier for us, because we can easily switch between animations/games of different users.

## Requirements
This "relay" between the client application and our lamp controlling server should be able to handle enough* concurrent connections sending very low resolution (28x14) but high refresh rate video streams while not adding a significant amount of latency such that real time games can be played without lags.

*enough means that the software should not bottleneck our hardware in any non-avoidable way, we expect up to hundreds of users testing their applications simultaneously

## Usage
The server will primarily be used for the already mentioned video streams.
Additionally it will be used for different events (keyboard, gamepad, buttons, etc.) and everything we haven't thought about yet.
The generic implementation allows us to write all kinds of new applications that can be connected to each other through the server more easily.
Applications and ideas include:
- an animation manager to control other running animations
- a service that captures and stores all updates on a resource (e.g. recording an animation)


## Overview
We built a generic server that is roughly based upon the idea of HTTP (similar methods and the notion of resources) but also allows for publish-subscribe (PubSub) on resources as well as other features. For PubSub to work, we need Server-initiated communication in order to push updates to the clients, hence the need for bidirectional connections such as WebSockets and TCP.  
The resources are organized like files in a filesystem - a hierarchical directory tree.
The architecture mainly consists of different endpoints, request handlers, a directory datastructure and resources.

### Endpoints
An endpoint can be any kind of way that a request comes into the system.
We focus on WebSockets first but plan to also implement TCP, UDP, UNIX-Sockets and more.
The endpoint deserializes a request and passes it to all handlers that it is configured with.

#### Serialization
We use MessagePack for serialization of our own Protocol.

### Handlers
A handler is called by an endpoint and handles the request by first authenticating and checking authorization of the request (using an auth component) and then carrying out what the request is supposed to do, such as:  
- creating a new resource (POST)
- modifying an existing resource (PUT)
- deleting a resource (DELETE)
- retrieving the content of a resource (GET)
- subscribing/unsubscribing to all updates on a resource (STREAM/STOP)
- linking/unlinking resources (LINK/UNLINK)  
- and more (see Protocol)

The handler retrieves the resources to act upon from the directory, which is accessed using a path as in a filesystem.

### Directory
The directory is a simple tree that provides the creation, deletion and lookup of paths.
While the nodes of the tree are directories, the leaves contain the resources (which can also be thought of as files).

### Resource
A Resource has a current state, which can be updated and retrieved. Furthermore, it allows for the publish-subscribe pattern on its content.
How this is implemented might vary.

One solution is the use of a broker thread which receives a value, updates the state and sends the new value to all subscribers (the idiomatic Go style).
Another idea might be to eliminate the broker thread and let the client handling thread directly send the new value to all subscribers, but this might result in higher latencies (needs benchmarks for verification).

Another solution is a really interesting thread-safe queue implementation with multiple read ends.
The implementation of this approach is based on originates from the implementation of "Control.Concurrent.Chan" in the Haskell language.
This concept can also be found in Apache Kafka where consumer groups can have different offsets.

### Protocol
#### Binary Serialization with MessagePack
Our protocol relies on MessagePack for binary serialization instead of plain text formats such as JSON or XML or other binary formats such as protobuf. We opted for a binary format because it allows us to include binary data (without the need for base64 encoding) and the encoded size of an object is much smaller compared to plain text formats.  
There are MessagePack libraries for many programming languages that can be found on the official website https://msgpack.org/.  

#### Protocol Schema
The request and response are MessagePack map types, containing (or requiring) the following entries.  
None of the map entries are optional, but may be null or otherwise empty.  
An asterisk `<*>` indicates that any valid MessagePack type is accepted as a type.
##### Request
```
{
    REID: <*>,                  # client chosen id to identify which response corresponds to this request
    AUTH: <Map<String,*>>,      # credentials containing {"USER": "YourUsername", "TOKEN": "Your-API-Token"}
    VERB: <String>,             # the request method (see VERB)
    PATH: <String[]>,           # the path of the resource that is the target of this request
    META: <Map<*,*>>,           # meta information about the request or additional parameters
    PAYL: <*>                   # optional payload for sending data to the server
}
```

##### Response
```
{
    REID: <*>,                  # the response contains the same REID that was sent with the request
    RNUM: <Int>,                # the response code (based on HTTP status codes)
​    RESPONSE: <String>,         # the response text
​    META: <Map<*,*>>,           # meta information about the response
    PAYL: <*>                   # optional payload for sending data to the client
    WARNINGS: <*[]>             # list of warnings for the client to notice (might as well contain detailed error messages)
}
```

#### Request Methods (VERBS)

##### POST
Combines CREATE (if not exists) and PUT
1. Creates a resource at the path if it does not already exist
(Missing parent directories in the path are created as well)
2. Updates the resources content with the payload (same as PUT)
- requires CREATE and WRITE permission

##### CREATE
Creates a resource at the path
- Missing parent directories in the path are created as well
- requires CREATE permission

##### MKDIR
Creates a directory at the path
- Missing parent directories in the path are created as well
- requires CREATE permission

##### DELETE
Deletes a resource or directory
- all active streams on the resource or resources that are contained in the directory are closed
- requires DELETE permission

##### LIST
Lists the directory tree starting from the given path (must be a directory)
- an empty path lists the whole tree from the root directory
- requires READ permission

##### GET
Returns the current content of the resource at the path inside the response payload
- requires READ permission

##### PUT
Updates the resource at the path with the contents of the payload
- requires WRITE permission

##### STREAM
Streams (subscribes to) the resource at the path and returns the current content of the resource as well as future updates to the resource inside the response payload
- multiple STREAM request from the same client to the same resource won't create another stream subscription
- responses sent as a result of resource updates contain the same REID as the initial STREAM request
- requires READ permission

##### STOP
Stops an active stream on the resource at the path
- only works if there is an active stream on the resource
- requires no permission

##### LINK
Links a destination resource (at the path) to a source resource
- the payload is interpreted as the path to the source resource
- will not succeed if a cyclical link is detected
- requires WRITE permission on the destination and READ permission on the source

##### UNLINK
Removes the link of a destination resource (at the path) to the source resource
- will not succeed if the link does not exist
- requires WRITE permissions on the destination