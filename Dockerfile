FROM golang:1.25.7-bookworm AS builder

WORKDIR /app/welfare-backend

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY welfare-backend/go.mod welfare-backend/go.sum ./
RUN go mod download

COPY welfare-backend ./
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /app/welfare-backend/welfare-backend ./cmd/server/main.go

FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/welfare-backend/welfare-backend /app/welfare-backend

EXPOSE 8080

CMD ["/app/welfare-backend"]

