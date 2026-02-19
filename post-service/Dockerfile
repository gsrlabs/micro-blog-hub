# ---------- Build stage ----------
FROM golang:1.25-alpine AS builder
WORKDIR /app
RUN apk add --no-cache gcc musl-dev git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app ./cmd/app

# ---------- Final stage ----------
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/app .
COPY config/ config/
EXPOSE ${POST_SERVICE_APP_PORT:-8050}
CMD ["./app"]
