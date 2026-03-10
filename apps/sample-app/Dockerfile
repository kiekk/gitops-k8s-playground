FROM golang:1.22-alpine AS builder

WORKDIR /build

COPY go.mod ./
COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app .

FROM alpine:3.19

RUN apk --no-cache add ca-certificates \
    && addgroup -S appgroup \
    && adduser -S appuser -G appgroup

USER appuser

COPY --from=builder /app /app

EXPOSE 8080

ENTRYPOINT ["/app"]
