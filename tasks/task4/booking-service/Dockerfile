# Используем официальный минимальный образ Go
FROM golang:1.21-alpine

# Ожидаемый код:
RUN apk add --no-cache curl

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем Go-файл и скрипт проверки DNS
COPY main.go .
COPY check-dns.sh .

# Ожидаемый код:
RUN chmod +x check-dns.sh

# TODO: Собрать бинарник из main.go
# Ожидаемый код:
RUN go build -o booking-service main.go

EXPOSE 8080

# Ожидаемый код:
CMD ["./booking-service"]
