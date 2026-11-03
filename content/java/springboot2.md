---
title: "springboot2"
folder: "Java"
created: "2026-03-24T07:34:19.000Z"
modified: "2026-04-17T07:31:49.000Z"
note_id: "x-coredata://8788A586-C8FC-4484-AB68-760AF85E0359/ICNote/p3543"
source: "apple-notes"
---

3 low lvl spring messaging tech projects mentioned
apache kafka, spring amqp, apache pulsar

kafka grp id in consumer config represents an exclusive consumer grp that prevents message duplication across diff partitions

@KafkaListener
annotation used to listen for kafka events in spring application

problems with java messaging service(JMS)
it allows broker upgrades that can break wire protocol, forcing both producer and consumer to upgrade simultaneously
secondly, producer and consumer share the same addr, making routing b/w them difficult

kafkaTemplate method is used in spring to send kafka message(specified topic with a key and payload)

decoupling allows disconnection of two or more previously linked systems, allowing them to func independently

unique char of the kafka docker image
compiled as a graalvm native image

two key properties needed for config kafka serialization are: key serializer and value serializer, which handle converting java obj to network wire protocol for kafka transmission

config property reqd to trust json deserialization packages is consumer.setProperty(‘spring.json.trusted.packages’, ‘\*’)

web stack dependency provides json marshalling support, which is necessary for converting java obj to and from json format.
json marshalling is the process of converting programming language data structure(such as structs, maps, or objs) into a json formatted string or byte slice.

four main integration styles
file transfer, shared database, rpc(remote procedural call), and messaging

adv of messaging, provides asynchronous by default, provides decoupling, allows systems do operate independently, enables file and forget communication, and does not require systems to be in the same place at the same time.

3 main components of messaging workflow
inbound adapter(source of events), processing pipeline(filtering, transforming, routing), and outbound adapter(destination for processed messages)

disadv of rpc
synchronous method invocations, systems must be operational simultaneously, high coupling, and potential failure if the remote service becomes unavailable.

architectural pattern that describes the messaging integration approach is pipe and filters architecture.

inbound adapter is a component that pulls data from an external source, such as a file system, database, or messaging queue, and introduces it into an integration flow. in this example, the file inbound adapter, reads files from a specified memory.

working of polling in spring integration when using an inbound adapter
spring integration polls an external source every second. the polling interval can be customized using fixed rate, fixed delay, or cron experience. this approach allows controlled and serialized message processing in a staged event driven architecture.

transformers in integration flow
they can convert messages from one format to another, such as transforming a file to a string(FileToStringTransformer) or converting a JSON string to a Java object(JSONToObjectTransformer). They enable data type conversion and preparation for further processing in the integration flow.

message splitting in spring integration
message splitting is a technique where a single message is broken down into multiple messages. in this example, a purchase order with multiple line items is split into individual line item messages, which can be processed separately and then potentially aggregated back together.

message headers in spring integration
message headers are metadata associated with a message payload, providing additional context and info abt the message. they can include details like file name, message origin, correlation info for splitting and aggregation, and other out-of-band info not part of the main proj.

message channel in springboot
a message channel is a logical concept representing a pipe through which messages, flow allowing for the transmission of data b/w diff parts of an integration flow.

by using a shared message channel, diff sources like a file inbound adapter or a REST controller can send messages into the same channel, which is then processed by integration flow

by default, the dispatch from one component to another in the same integration flow is synchronous, meaning all steps are executed in the same thread

error handling in springboot
spring integration creates a default error channel that can be listened to, allowing for error handling, redelivery, or custom error processing strategies

potential ways to improve the message channel
message channels can be backed by a database, have messages written to SQL tables, include security filters, and can be configured to be synchronous or asynchronous

@Transactional
it ensures that database operations within the method are executed as a single atomic transaction, providing data consistency and integrity

ListCrudRepository interface: provides CRUD operations for an entity without having to manually implement these methods, both read and write CRUD operations

@Service
it marks a java class as a service layer component, allowing spring to automatically detect and manage the class as a spring bean

primary benefit of packaging related classes in the same package;
reduced code visibility

@PathVariable
this annotation is used to mark a method that can be called via specific HTTP endpoint

four messaging and integration styles
even notification, event carried state transfer, event sourcing, cqrs(command query responsibility segregation)

event notification
telling one system abt the fact of a change without providing additional details

even carried state transfer
a messaging approach where each event message contains actionable details that the consumer can use, avoiding coupling b/w the producer and consumer by including all necessary info within the message itself

what is the use of messages
message helps in system reconstruction and provides an audit log of all system events.

even sourcing
a pattern where every change to the system state is captured as a sequence of events, allowing for system reconstruction, historical replay, and the ability to apply new algo retroactively to past events

cqrs(command query responsibility segregation)
it represents the concept of using diff data structures optimized for diff read and write modalities, such as using postgres for writes, redis for fast reads, and elasticsearch for full text search

outbox pattern
it helps reconcile non transactional state against a transactional ledger, ensuring event delivery and consistency by storing events in a database table and guaranteeing their eventual processing, even if initial delivery fails.

when events are published synchronously, they occur in the same thread, which means if one component takes a long time to process(using Thread.sleep()), it blocks the entire request and can cause significant delays in response times.

@Async annotation
it wraps the method invocation in a separate thread, allowing the original method to return immediately while the event is processed in the background, providing immediate responsiveness but potentially risking state loss

the event publication table stores the event’s ID, listener ID, event type, serialized event representation, publication date, and completion date, which helps track the status and processing of events

using lock registry(such as JDBC or Mongo lock registry) allows only one node in a cluster to acquire a lock and run a specific piece of code, preventing multiple nodes from republishing the same events simultaneously

when listener fails to complete the outbox pattern; the event is automatically retired on JVM restart

implement idempotent code that can safely handle repeated event processing to handle potential duplicate event processing in distributed systems

messaging modules that support externalization(allows communication b/w components in diff JVMs), which can publish messages to diff messaging system like kafka, rabbitMQ, SQS, Pulsar, and other using spring integration

ArchUnit, which allows testing and enforcing architectural rules and constraints in the system

when a nested package type is injected into another module, the test will fail, indicating that an implementation detail of one module has been inadvertently leaded to another module

PlantUML diagram that can checked into version control and rendered using tools like Graphviz

spring modulus helps developers achieve decoupled and modular code that can be tested in their application architecture

server.port = 0 in application.properties file means start on random port; automatically selects an unused port

spring cloud is a set of extensions for building distributed systems and microservices layered on top of springBoot, designed to make building complex distributed systems easier

8888 is the default port for the spring cloud config server

a service registry helps advertise available services, manage dynamic service locations, and provide more reliable alternative to DNS for discovering and tracking microservices
in spring boot, it can also inferred as a centralized database or directory that stores the network locations(IP addr and port) of all active microservices instances in a distributed system.

problem solved by unique instance id in a service registry
unique instance id allows distinguishing b/w multiple instances of the same service running on the same host, preventing registration conflicts and enabling proper service tracking.

dns is considered problematic for microservice systems
dns encourages caching, which is bad for dynamic cloud’s native systems

mospring cloud gateway used for in microservices arch
spring cloud gateway acts as a micro proxy, facilitating routing and load balancing between services

key components required to build load balancer
u need a REST controller, a load balanced REST client, and the ability to use a logical service name registered in the service registry

how does a load balanced REST client communicate with downstream services
it first queries the service registry to discover available service instances, then uses a load balancing algo to distribute requests across those instances

‘logical name’ in the context of service discovery
logical name is an identifier use to refer to a service in the service registry, which is not a physical hostname or DNS addr

RestTemplateBuilder is the spring cloud component is used to create a load balanced REST client

spring cloud gateway is a microproxy that acts like nginx, allowing proxying of endpoints, load balancing, and request/response filtering. it can redirect requests to diff services, strip path prefixes, and handle routing at the gateway lvl

router functions represent an alternative component model for defining endpoints using functional programming techniques. instead of using annotations like @Controller, router functions use predicates and lambda expressions to define routes and handle requests

adv of using gateway provide for handling CORS and authentication
a gateway can serve both frontend applications and backend APIs from the same host and port, effectively eliminating CORS issues. it can also centralize authentication handling, such as processing JWT tokens, which simplifies frontend security implementation

spring cloud provides capabilities like centralized config, service registration and discovery, client-side load balancing, and a flexible gateway for routing and filtering requests across microservices

‘rewrite path’ filter in spring cloud
to strip away a specific prefix from the request path
