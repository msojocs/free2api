FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 CGO_CFLAGS="-D_LARGEFILE64_SOURCE" GOOS=linux go build -o ai-auto-register-server ./cmd/

FROM alpine:3.19

RUN apk add --no-cache ca-certificates sqlite-libs tzdata

WORKDIR /app
COPY --from=builder /app/ai-auto-register-server .

RUN mkdir -p /app/data

EXPOSE 8080

CMD ["./ai-auto-register-server"]
