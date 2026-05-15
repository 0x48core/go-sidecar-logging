# Stage 1: build
FROM golang:1.25-alpine AS builder

WORKDIR /src

# download deps first — cached unless go.mod/go.sum change
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/transactions-api .

# Stage 2: minimal runtime image
FROM alpine:3.19

RUN apk --no-cache add ca-certificates && \
    mkdir -p /app/logs

COPY --from=builder /app/transactions-api /app/transactions-api

EXPOSE 8080

ENTRYPOINT ["/app/transactions-api"]