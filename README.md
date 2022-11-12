# rowix

A collaborative text editor written in Go.

## Usage

To start the server:

```
go run server/main.go
```

To start the client:

```
go run client/*.go
```

(spin up at least 2 clients - it's a collaborative editor! Also works with a single client.)

## How does it work?

Here's a basic explanation:

- Each client has a CRDT-backed local state.
- The CRDT has a `Document` which can be represented by a sequence of characters with some attributes.
- The server is responsible for:
  - establishing connections with the client
  - maintaining a list of active connections
  - broadcasting operations sent from a client to other clients
- Clients connect to the server and send operations to the server.
- The TUI is responsible for:
  - Rendering document contents
  - Handling key events
  - Generating payload on key presses
  - Dispatch generated payload to the server
