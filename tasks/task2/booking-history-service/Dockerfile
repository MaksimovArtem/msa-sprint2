FROM golang:latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go mod tidy
RUN go build -o booking-history-service ./cmd/server

CMD ["./booking-history-service"]

