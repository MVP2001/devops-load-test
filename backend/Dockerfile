FROM golang:1.21-alpine AS builder

# Устанавливаем необходимые пакеты для SQLite
RUN apk add --no-cache gcc musl-dev sqlite-dev build-base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Собираем с правильными флагами для Alpine
RUN CGO_ENABLED=1 GOOS=linux CGO_CFLAGS="-D_LARGEFILE64_SOURCE" go build -o server ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /root/

COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]
