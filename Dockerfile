FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" \
    -o /bin/fence ./cmd/fence/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/fence /usr/local/bin/fence
ENV PORT=8770 \
    DATA_DIR=/data \
    FENCE_ADMIN_KEY="" \
    FENCE_ENCRYPTION_KEY=""
EXPOSE 8770
ENTRYPOINT ["fence"]
