FROM golang:latest

ENV http_proxy="http://172.18.105.61:9999/"
ENV https_proxy="http://172.18.105.61:9999/"


# ENV http_proxy="http://172.18.105.154:9999/"
# ENV https_proxy="http://172.18.105.154:9999/"

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go mod tidy
RUN go build -o booking-service ./cmd/server

EXPOSE 9090

CMD ["./booking-service"]

