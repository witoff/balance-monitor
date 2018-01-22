FROM golang:1.9.2 as builder
WORKDIR /go/src/github.com/witoff/balance_monitor/
COPY main.go .
RUN go get github.com/sendgrid/sendgrid-go && \
    go get github.com/sendgrid/sendgrid-go/helpers/mail && \
    go get gopkg.in/yaml.v2
RUN CGO_ENABLED=0 GOOS=linux go build -a -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
RUN mkdir -p /opt/balance_monitor
WORKDIR /opt/balance_monitor
COPY --from=builder /go/src/github.com/witoff/balance_monitor/main .
COPY config.yaml .
CMD ["./main"]
