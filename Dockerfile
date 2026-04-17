FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /forager ./cmd/forager

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /forager /usr/local/bin/forager
COPY configs/config.yaml /etc/forager/config.yaml
EXPOSE 8080
ENTRYPOINT ["forager", "--config", "/etc/forager/config.yaml"]
