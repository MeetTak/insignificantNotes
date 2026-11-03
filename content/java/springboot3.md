---
title: "springboot3"
folder: "Java"
created: "2026-04-15T05:34:21.000Z"
modified: "2026-04-24T19:22:41.000Z"
note_id: "x-coredata://8788A586-C8FC-4484-AB68-760AF85E0359/ICNote/p3600"
source: "apple-notes"
---

spring security acts as a big filter around the application that guards requests from the outside world, handling authentication(identifying who is making the request) and authorization(determining what permissions the authenticated user has)

Principle is a security obj that primarily provides the name of the currently authenticated user, and has been part of the java security specification for approximately 20 yrs

webAuthn core dependency, which is intended to improve authentication mechanisms beyond traditional password-based systems

two key aspects of security
authentication(identifying who is making the mistake)
authorization(determining what permissions they have once inside the system)

problem with password based security
they are inherently broken and not a good way to build a secure system

getName() method can be used to get the name of the currently authenticated principal

password encoder in spring security
to securely encode passwords so they are never stored in plain, text, support password migration, and enable different encoding methods like bcrypt for enhanced security

significance of password prefix in spring security’s password encoding
the prefix indicates the encoding method used(eg bcrypt), which allows for supporting multiple password encoding schemes and enables seamless migration b/w diff encoding techniques

sha256 is considered as insecure hashing method as it can be easily reversed with relatively low computing power, making vulnerable to potential password breaches and not providing sufficient security for storing user credentials

passkeys are a webAuthn-based authentication method that uses public key cryptography, allowing users to log in using biometric factors like face ID or touch ID, stored securely in a trusted agent like a browser, operating system, or password manager

webAuthn provides a more secure and user-friendly authentication method by leveraging public key cryptography and allowing users to authenticate using trusted devices and biometric factors like face ID or touch ID, eliminating the need for traditional passwords

OAuth specification includes OAuth authorization servers, OAuth resource servers, and OAuth clients

OAuth allows services to authenticate users without storing their username and password, providing limited and controlled access to user resources

OAuth grant represents the specific permissions a client is allowed to have on a user’s behalf

a token relay filter ensures that if no authentication token exists, the request is rejected and redirected to the authorization server for authentication

what happens when a OAuth client successfully authenticates a user
the user is redirected back to the client application with a token that can be used to access specific APIs on the user’s behalf

primary purpose of an OAuth authorization server
to centralize authentication and token validation for multiple applications

primary advantage of using OAuth for authentication
the application never receives the user’s actual username and password

different kinds of communication protocol: webSocket, gRPC, HTTP, GraphQL

concurrentHashMap in the context of chat memory
to create a multi-tenant system where each user gets their own advisor and chat memory, allowing independent conversation tracking for diff users

three primary methods that must be implemented in a custom chat memory interface
void add, list of messages get, and clear

context window
a measure of the complexity of the request or response, calculated in tokens, which determines the amount of information that can be processed in a single interaction

vector store is a database optimized for semantic similarity search, which turn data like strings, images, or audio into numerical embeddings that can be compared using mathematical techniques like cosine similarity

embeddings in the context of data processing
embeddings are numerical representations of data(like strings or images) converted into a matrix or array of numbers, which allows for mathematical comparison and semantic similarity analysis

technologies that support vector stores, including PGVector, Milvus, Pinecone, Weaviate, Neo4j, MariaDB, Oracle, Redis, Elasticsearch, MongoDB, and others.

creating embeddings is computationally expensive, often requiring specialized processing, which is why GPUs have become popular for handling such matrix multiplication and computational tasks.

data can be converted into embeddings through local machine processing or by using external services like openAI’s REST endpoint, which transforms input data into a numerical matrix representation

embedding is the process turns data into a bunch of numbers

ToolParam is the annotation used to provide description for tool parameters

using virtual threads is recommended to handle network I/O efficiently, improving performance and resource utilization when implementing network communication protocols like MCP.

initial communication method used by mcp is standard input/output protocol

anthropic company first announced the model context protocol
