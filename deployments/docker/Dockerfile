FROM golang:1.26.10 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/seed ./cmd/seed

FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=builder /tmp/api /app/api
COPY --from=builder /tmp/seed /app/seed
EXPOSE 8080
ENTRYPOINT ["/app/api"]
