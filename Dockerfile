FROM golang:1.18-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY ./ ./

RUN apk add --no-cache bash

RUN go build -o /rowix-server ./server/main.go

EXPOSE 8080

CMD [ "/rowix-server -addr rowix-server.fly.dev" ]
