![rowix banner](.github/assets/rowix.png)

# rowix

A collaborative text editor written in Go.

![Preview](.github/assets/demo.gif)

## Usage

The easiest way to get started is to download `rowix` from the [releases](https://github.com/burntcarrot/rowix/releases).

Then start a server:

```
Usage of rowix-server:
  -addr string
        Server's network address (default ":8080")
```

Then start a client:

```
Usage of rowix:
  -login
        Enable the login prompt
  -path string
        Server path (default "/")
  -server string
        Server network address (default "localhost:8080")
  -wss
        Enable a secure WebSocket connection
```

### Local setup

To start the server:

```
go run server/main.go
```

To start the client:

```
go run client/main.go
```

(spin up at least 2 clients - it's a collaborative editor! Also works with a single client.)

## Deployment

The easiest way to deploy would be use to [fly.io](https://fly.io/).

[fly.io](https://fly.io/) has an [amazing guide](https://fly.io/docs/hands-on/) to get started!

Here's the `Dockerfile`:

```Dockerfile
FROM golang:1.18-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY ./ ./

RUN apk add --no-cache bash && go build -o ./rowix-server ./server/main.go

EXPOSE 8080

CMD [ "/app/rowix-server" ]
```

Here's a sample `fly.toml` file:

```toml
app = "rowix-server-<YOUR_NAME>"
kill_signal = "SIGINT"
kill_timeout = 5
processes = []

[env]

[experimental]
  allowed_public_ports = []
  auto_rollback = true

[[services]]
  http_checks = []
  internal_port = 8080
  processes = ["app"]
  protocol = "tcp"
  script_checks = []
  [services.concurrency]
    hard_limit = 25
    soft_limit = 20
    type = "connections"

  [[services.ports]]
    force_https = true
    handlers = ["http"]
    port = 80

  [[services.ports]]
    handlers = ["tls", "http"]
    port = 443

  [[services.tcp_checks]]
    grace_period = "1s"
    interval = "15s"
    restart_limit = 0
    timeout = "2s"
```

## How does it work?

Here's a basic explanation:

- Each client has a CRDT-backed local state (document).
- The CRDT has a `Document` which can be represented by a sequence of characters with some attributes.
- The server is responsible for:
  - establishing connections with the client
  - maintaining a list of active connections
  - broadcasting operations sent from a client to other clients
- Clients connect to the server and send operations to the server.
- The TUI is responsible for:
  - Rendering document content
  - Handling key events
  - Generating payload on key presses
  - Dispatching generated payload to the server

## License

`rowix` is licensed under the [MIT license](LICENSE).

## Future plans

Text editors is a huge space - there's lots of improvements to be made!
