FROM golang:1.22.5-alpine AS builder

ENV CGO_ENABLED=1
WORKDIR /app

RUN apk add --no-cache --update git build-base
COPY . .
RUN go mod tidy && \
    go build -o mails cmd/server/main.go

FROM golang:1.22.5-alpine

ENV TZ=UTC
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata libc6-compat libgcc libstdc++
COPY --from=builder /app/mails .

ENTRYPOINT ["/app/mails"]
