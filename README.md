# The Lighthouse Server (Beacon)
### **B**asis for **E**very **A**pplication with **C**onnections **O**ver **N**etworks

## About
At Project Lighthouse we wanted to provide our users (mostly students) with a platform for building animations and games to be played on our big screen, the "Lighthouse" - an LED installation inside a 14 story university building.  
Because users need to test their applications against our API, we needed a kind of virtual Lighthouse for that.  
Such a thing would also make management during events/shows much easier for us, because we can easily switch between animations/games of different users.

## Requirements
This "relay" between the client application and our lamp controlling server should be able to handle enough* concurrent connections sending very low resolution (28x14) but high refresh rate video streams while not adding a significant amount of latency such that real time games can be played without lags.

*(enough means that the software should not bottleneck our hardware in any non-avoidable way, we expect up to hundreds of users testing their applications simultaneously)

## Usage
The server will primarily be used for the already mentioned video streams.
Additionally it will be used for different events (keyboard, gamepad, buttons, etc.) and everything we haven't thought about yet.
The generic implementation allows us to write all kinds of new applications that can even be connected through the server more easily.
Applications and ideas include:
- an animation manager to control other running animations
- a service that captures and stores all updates on a resource (e.g. recording an animation)


## Overview
We built a generic server that is roughly based upon the idea of HTTP (similar methods and the notion of resources) but also allows for publish-subscribe (PubSub) on resources.
The resources are organized like files in a filesystem - a hierarchical directory tree.
The architecture mainly consists of endpoints, handlers, directory and resources.

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

The handler retrieves the resources to act upon from the directory, which is accessed using a path as in a filesystem.

### Directory
The directory is a simple general tree that provides the creation, deletion and lookup of paths.
While the nodes of the tree are directories, the leaves contain the resources (which can also be thought of as files).

### Resource
A Resource has a current state, which can be updated and retrieved. Furthermore, it allows for the publish-subscribe pattern on its content.
How this is implemented might vary.

One solution is the use of a broker thread which receives a value, updates the state and sends the new value to all subscribers (the idiomatic Go style).

Another solution is a really interesting thread-safe queue implementation with multiple read ends.
The implementation of this approach is based on originates from the implementation of "Control.Concurrent.Chan" in the Haskell language.
This concept can also be found in Apache Kafka where consumer groups can have different offsets.