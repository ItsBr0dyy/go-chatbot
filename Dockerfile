FROM golang:1.25-alpine
RUN apk add --no-cache git ca-certificates tzdata
ENV GOTOOLCHAIN=auto
WORKDIR /app
CMD ["sh", "-c", "go build -o /tmp/bot . && exec /tmp/bot"]