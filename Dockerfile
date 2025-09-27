# Stage 1: Build
FROM golang:1.24-alpine AS builder
# arguments
ARG APP_NAME=default-backend-service
ENV APP_NAME=${APP_NAME}
WORKDIR /app
RUN apk add --no-cache git

# Chỉ copy go.mod và go.sum trước để cache dependency
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -ldflags="-s -w" -o ${APP_NAME} ./cmd/server/

# Stage 2: Runtime
FROM alpine:latest
# Cài CA certificates
RUN apk --no-cache add ca-certificates
ARG APP_NAME=default-backend-service
ENV APP_NAME=${APP_NAME}
WORKDIR /root/
COPY --from=builder /app/${APP_NAME} .
COPY --from=builder /app/config/*.yaml .
COPY --from=builder /app/docs ./docs

EXPOSE 8080
CMD sh -c "./${APP_NAME}"