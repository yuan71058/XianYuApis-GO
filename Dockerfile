FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /xianyuapis ./cmd/demo/

FROM alpine:3.19
RUN apk add --no-cache nodejs
COPY --from=builder /xianyuapis /usr/local/bin/xianyuapis
COPY assets/ /app/assets/
WORKDIR /app
ENTRYPOINT ["xianyuapis"]
