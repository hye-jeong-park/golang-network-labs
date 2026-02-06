FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy
RUN go build -o tcp-server .

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/tcp-server /app/tcp-server
EXPOSE 9000
CMD ["/app/tcp-server"]