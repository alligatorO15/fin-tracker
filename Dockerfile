# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o fintracker ./cmd/server

# Final stage
FROM alpine:3.19

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

ENV TZ=Europe/Moscow

COPY --from=builder /app/fintracker .

EXPOSE 8080

CMD ["./fintracker"]