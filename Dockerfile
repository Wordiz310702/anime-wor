# rebuild: 2026-09-21-1
# ---- Этап 1: сборка ----
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o anime-wor .

# ---- Этап 2: рантайм ----
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/anime-wor .
COPY --from=builder /app/public ./public

RUN mkdir -p /data
RUN mkdir -p /app/public/uploads

EXPOSE 80

CMD ["./anime-wor"]