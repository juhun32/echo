FROM golang:1.24.3-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/main ./cmd

FROM alpine:3.20
WORKDIR /app
RUN apk --no-cache add ca-certificates

COPY --from=builder /out/main /app/main

EXPOSE 8080
CMD ["/app/main"]