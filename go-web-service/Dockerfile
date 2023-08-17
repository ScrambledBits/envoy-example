FROM golang:1.21.0-alpine3.18

WORKDIR /app

COPY main.go .

RUN go env -w GO111MODULE=auto && go build -o main .

EXPOSE 8080

CMD ["/app/main"]
