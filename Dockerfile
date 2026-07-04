# Stage 1 — builder
FROM golang:1.26.4-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o agency-service .

# Stage 2 — final image
FROM gcr.io/distroless/static-debian12

WORKDIR /

COPY --from=builder /app/agency-service .
COPY migrations/ migrations/

EXPOSE 8082

ENTRYPOINT ["/agency-service"]