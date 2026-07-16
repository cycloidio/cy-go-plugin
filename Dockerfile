FROM golang:1.25.0 AS builder

WORKDIR /plugin

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 go build -o /my_app

FROM alpine:3.21
WORKDIR /plugin
COPY --from=builder /my_app /my_app
COPY --from=builder /plugin/*.yaml ./
COPY --from=builder /plugin/*.sql ./
EXPOSE 8080
CMD ["/my_app"]
