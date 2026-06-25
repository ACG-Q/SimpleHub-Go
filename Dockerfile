# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache nodejs npm

WORKDIR /app

# Copy frontend source and build
COPY web/ ./web/
RUN cd web && npm ci && npm run build

# Copy Go dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build Go backend
COPY . .
RUN cp -r web/dist cmd/server/dist && \
    CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/server

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=builder /app/server .
# Optional: custom frontend can be mounted at /app/web/dist
RUN mkdir -p /app/web/dist /app/data && chown -R app:app /app

USER app

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/ || exit 1

ENV PORT=3000
ENV DATABASE_URL=file:/app/data/db.sqlite

CMD ["./server"]
