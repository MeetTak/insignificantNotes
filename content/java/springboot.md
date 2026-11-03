---
title: "springBoot"
folder: "Java"
created: "2026-02-10T17:40:19.000Z"
modified: "2026-03-24T07:34:03.000Z"
note_id: "x-coredata://8788A586-C8FC-4484-AB68-760AF85E0359/ICNote/p3486"
source: "apple-notes"
---

two general files that can be used for database initialization
schema.sql and data.sql

flyway is the tool used for data migration

two sql files that spring boot use for the database intialization
schema.sql and data.sql

config property to ensure sql intialization scripts run with a non embedded database like postgresql
spring.sql.init.mode=always

flyway prevent re-running the same migration script; by computing and storing a hash of the migration script, and preventing reapplication if the hash changes

spring data is an umbrella project for orm that provides modules for different database like sql, jdbc, jpa, neo4j, mongodb, redis, and others, offering consistent approach to data persistence across various data stores.

rowmapper job is to convert a resultSet from a sql database query into domain objects by mapping each row of the result set to a specific obj in the application domain model.

spring batch is a framework for building etl(extraction, transformation, loading) and batch processing applications.

a chunk in spring batch represents a block of data processed together in a single transaction. The chunk size determines how many records are read, processed, and written together. If an error occurs during processing, the entire chunk will be rolled back, ensuring data integrity and providing a way to handle potential issues in batch processing.

ItemReader is responsible for reading data from a source one item at a time, while ItemWriter writes a chunk of items to a destination.

Spring Batch supports distributed processing through techniques like partitioned steps and remote chunking. These methods allow work to be distributed across multiple nodes in a cluster. Nodes can process data ranges or actual data chunks, enabling concurrent processing across 100 different nodes, which helps scale batch processing applications horizontally.

main components of spring batch
job, step, reader, writer, and processor

parameterized type reference, it is used to overcome java limitation of type erasure, where generic type info is lost at runtime.

to create declarative http client interface in spring, use the @HttpExchange annotation

bean qualifiers help spring disambiguate between multiple beans of the same type by allowing you to specify a unique name or annotation.

problem with java generic is that at runtime, generic type info is erased.

to create custom qualifier annotation, start by creating a meta-annotation by composing the @Qualifier annotation with a custom annotation. this allows u to create type-safe, readable qualifiers that can be used to disambiguate beans without relying on magic strings.

@GetMapping annotation used to specify an HTTP GET request in spring mvc

@RequestMapping is the lowest lvl annotation for http request mapping in spring mvc

hypermedia links can improve application design by showing possible navigable states, simplifying code logic and preventing invalid state transitions.

representation model assembler in spring hateos
to contribute and assess links that are viable for a model, allowing reusable link generation across different parts of the code

benefit of entity model in spring hateos
to wrap resources with metadata and links, providing navigable and self descriptive json payloads

purpose of avoiding @ResponseBody in a spring mvc controller method
When @ResponseBody is not used, Spring MVC interprets the
return value as a view name and looks up a view using conventions
(e.g., searching in src/main/resources/templates for a matching
HTML file).

how does spring mvc resolve view name by default
Spring MVC uses a ViewResolver to look up views by convention,
typically searching in src/main/resources/templates directory with
the returned string as the filename, appending .html.

model.addAttributes() method is used to add attributes to the model, which can then be accessed in the view template.

thymeleaf is used for server side templating technologies in spring mvc, used for rendering dynamic content on the server side.

Spring MVC allows returning multiple views simultaneously by
returning a collection of ModelAndView objects or potentially a
collection of strings, enabling rendering of multiple view fragments
at once.

3 main verbs of graphql: queries, mutations, and subscriptions

problem solved by batch mapping in graphql
batch mapping prevents N+1 query problem by allowing efficient retrieval of related data in a single cell, instead of making separate calls for each obj.

graphql uses schema first design for defining data structure

gRPC is schema first, requiring developers to define their service interface using protocol buffers(.proto) before implementation.
gRPC uses HTTP/2 as its underlying transport protocol.
grpcurl command to call gRPC endpoints directly form the command line.
when implementing a gRPC service in spring, u just need to extend the generated base service class.

to initialize a database on spring startup create a file with a name schema.sql and define the table structure. spring boot will automatically create a database schema when the application starts.

virtual threads are beneficial for i/o operations, such as database queries, where a thread would typically wait for response. they allow more efficient handling of network requests by moving waiting operations off the main thread, enabling better scalability and resource utilization.

@id annotation signals the primary key of an entity across different spring data modules.

create a data.sql file with INSERT statements. spring boot will automatically run this file to populate the database with initial data when the application starts.

springboot actuator is an observability module designed to surface information about the state and health of an application, providing endpoints for monitoring metrics, health checks, configurations, and system information in production environments.

the /actuator/info endpoint can display custom, arbitrary information such as project such as project metadata, git commit details, build information, and any custom properties added by the developer.

the /actuator/health endpoint provides the overall health status of the application, indicating whether the service is healthy or not.

micrometer is a vendor-neutral application observability facade(eg frontend part for building) for the jvm. it provides a standard way to collect and expose metrics, and it can be used independently of spring. many libraries and drivers include micrometer to emit relevant metrics about their performance and state.

sbom(software bill of materials) is a manifest that provides detailed info about an application’s dependencies, including their origin, creators, and SHA(secure hash algo) values. it is crucial for security teams to understand the provenance of dependencies and ensure transparency in the software supply chain.

management.endpoints.web.exposure.include=\*
exposes all actuator endpoints for demonstration purposes

actuator enables applications to provide production-level operational features, giving the application the ability to ‘speak’ by exposing monitoring and management endpoints

command to build native image in a maven proj
maven skip tests -P Native native:compile
to build a native graalvm image that can be run in the environment

two ways to provide database connection config for a prod env

1. external application.properties file with the datasource config
2. use env variables to specify connection details like username, password, and url

docker compose is not reliable in production env bcoz it automatically provides credentials in development mode, but these credentials are not automatically available when running a production binary

adv of externalizing properties is that it allows u to deploy the same binary across diff env (development, QA, staging) by changing confg externally without modifying the binary itself

adv of using a native graalvm image
faster startup time and less ram usage

maven command can be used to build a native docker image for a spring boot application
mvn -Pnative spring-boot:build-image

env variables that need to be set before running a docker image
spring.datasource.url, spring.datasource.username, spring.datasource.password

host.docker.internal for reference the host machine from within the container, when running a docker container
