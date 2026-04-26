FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app ./cmd/bot

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app /app
COPY migrations/ /migrations/
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://localhost:8080/healthz || exit 1
ENTRYPOINT ["/app"]
