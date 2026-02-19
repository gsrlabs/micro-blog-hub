# ---------- Build stage ----------
FROM golang:1.25.5-alpine AS builder
WORKDIR /app
RUN apk add --no-cache gcc musl-dev git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app ./cmd/app

# ---------- Final stage ----------
FROM alpine:latest
ARG TIME_ZONE=UTC
ENV TZ=${TIME_ZONE}
RUN apk --no-cache add ca-certificates tzdata && \
	cp /usr/share/zoneinfo/${TZ} /etc/localtime && \
	echo ${TZ} > /etc/timezone
WORKDIR /app
COPY --from=builder /app/app .
COPY --from=builder /app/migrations ./migrations
COPY config/ config/
EXPOSE ${AUTH_SERVICE_APP_PORT:-8040}
CMD ["./app"]