FROM golang:1.22-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o todo .

FROM alpine:3.20

COPY --from=build /app/todo /usr/local/bin/app

ENTRYPOINT ["/usr/local/bin/app"]

