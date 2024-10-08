FROM golang:1.18-bullseye as builder
WORKDIR /go/src
COPY go.mod go.sum /go/src/
RUN go mod download
ADD . /go/src
RUN CGO_ENABLED=0 go build -buildvcs=false . && ls -l /go/src

FROM alpine:latest
WORKDIR /go
COPY --from=builder /go/src/web-realtime-api .
CMD ["./web-realtime-api"]

