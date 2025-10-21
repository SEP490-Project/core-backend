# ===============================================================
# Stage 1: Build
FROM golang:1.24-alpine AS builder

ARG APP_NAME=default-backend-service
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    PATH=$PATH:/go/bin \
    APP_NAME=${APP_NAME}

RUN apk add --no-cache busybox-static ca-certificates tzdata \
    && mkdir -p /bin \
    && cp /bin/busybox.static /bin/busybox \
    && /bin/busybox --install -s /bin

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download && go install github.com/swaggo/swag/cmd/swag@latest

COPY . .

RUN swag init -g ./cmd/server/main.go --output ./docs --parseInternal
RUN go build -ldflags='-w -s -extldflags "-static"' -a -o main ./cmd/server/main.go

# ===============================================================
# Stage 2: Final (scratch + minimal utilities)
FROM scratch AS final

ARG APP_PORT=8080
ENV APP_PORT=${APP_PORT} \
    TZ=Asia/Ho_Chi_Minh

# Copy passwd/group (so container can exec as non-root if needed)
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /bin /bin
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

WORKDIR /app

# Copy app binary + configs/docs
COPY --from=builder /build/main .
COPY --from=builder /build/config/ ./config/
COPY --from=builder /build/docs/ ./docs/

EXPOSE ${APP_PORT}

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD ["./main", "--health-check"] || exit 1

ENTRYPOINT ["./main"]
