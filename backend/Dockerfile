FROM golang:1.24.0 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o messageflow ./cmd/server
RUN mkdir -p /app/media

FROM gcr.io/distroless/base-debian12
WORKDIR /app

COPY --from=builder /app/messageflow /app/messageflow
COPY --from=builder /app/media /app/media

EXPOSE 8080

ENTRYPOINT ["/app/messageflow"]
