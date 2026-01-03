FROM golang:1.24.0 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o messageflow ./cmd/server

FROM gcr.io/distroless/base-debian12
WORKDIR /app

COPY --from=builder /app/messageflow /app/messageflow

EXPOSE 8080

ENTRYPOINT ["/app/messageflow"]
