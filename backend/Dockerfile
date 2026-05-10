# Dev image: source mounted at runtime, `go run` rebuilds on each compose-up.
# Slice 1.5 keeps things simple — production Dockerfile can come later.
FROM golang:1.25-alpine

RUN apk add --no-cache git

WORKDIR /app

# Cache deps separately from source.
COPY go.mod go.sum ./
RUN go mod download

EXPOSE 8080

CMD ["go", "run", "./cmd/levitate"]
