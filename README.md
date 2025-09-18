## Packages:

```bash
# Gorm related packages
go get gorm.io/gorm
go get gorm.io/driver/postgres
go get gorm.io/datatypes

# Core packages
go getgithub.com/google/uuid
go getgithub.com/spf13/viper
go getgithub.com/golang-jwt/jwt/v5

# Core OpenTelemetry SDK
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/sdk
go get go.opentelemetry.io/otel/sdk/log
# OTLP Log Exporter
go get go.opentelemetry.io/otel/exporters/otlp/otlplogs/otlplogshttp
# OpenTelemetry Zap logger bridge
go get go.opentelemetry.io/contrib/zapl


# Core HTTP Server packages
go get github.com/gin-contrib/cors
go get github.com/gin-gonic/gin
go get github.com/go-playground/validator/v10
go get github.com/gorilla/websocket
go get github.com/gin-contrib/cors
go get github.com/gin-contrib/cors
go get github.com/gin-contrib/cors

# Swaggo for auto generate Swagger documentation
go get github.com/swaggo/swag@latest
go get github.com/swaggo/gin-swagger
go get github.com/swaggo/files

# RabbitMQ packages
go get github.com/rabbitmq/amqp091-go

# Redis packages
go get github.com/redis/go-redis/v9
```

## Prerequisites:

- Before running the application, you need to have three services running:
  1. PostgreSQL
  2. RabbitMQ
  3. Redis or Valkey
- After that edit the `./config/config.yaml` file and set the correct values for the database, rabbitmq and redis connection strings.
- Install Swaggo before running the application through :
  ```bash
  go get github.com/swaggo/swag/cmd/swag@latest
  ```
- Generate the Swagger documentation through the following command:
  ```bash
  swag init -g ./cmd/server/main.go -o ./docs
  ```
- Because for some reasons of the swaggo packages, when the `docs/docs.go` file is generated, it will have these two lines about Delimiters, which will cause the build process to fail. Delete those two lines before running the application.
  ```go
  var SwaggerInfo = &swag.Spec{
      ...
  	LeftDelim:        "{{",
  	RightDelim:       "}}",
  }
  ```
- Finally, run the applicaiton through:
  ```bash
  go run ./cmd/server
  ```
