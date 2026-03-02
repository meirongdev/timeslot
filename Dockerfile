# ---- build stage ------------------------------------------------------------
FROM golang:1.25-alpine AS builder

# CGo is required by go-sqlite3
RUN apk add --no-cache gcc musl-dev

WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o /timeslot .

# ---- runtime stage ----------------------------------------------------------
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S timeslot && adduser -S -G timeslot timeslot

WORKDIR /app

# Copy binary and templates (templates/ must be alongside the binary at runtime)
COPY --from=builder /timeslot ./timeslot
COPY backend/templates/ ./templates/

# Data directory for SQLite volume mount
RUN mkdir -p /data && chown timeslot:timeslot /data

USER timeslot

EXPOSE 8080

ENTRYPOINT ["/app/timeslot"]
CMD ["-config", "/config/config.json"]
