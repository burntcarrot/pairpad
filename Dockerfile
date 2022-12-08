FROM golang:1.18-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY ./ ./

# skipcq: DOK-DL3018
RUN apk add --no-cache bash && go build -o ./rowix-server ./server/main.go

EXPOSE 8080

CMD [ "/app/rowix-server" ]
