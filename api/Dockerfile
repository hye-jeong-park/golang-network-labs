FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy
RUN go build -o api-server ./cmd/api-server

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/api-server /app/api-server
EXPOSE 8080 8081
CMD ["/app/api-server"]