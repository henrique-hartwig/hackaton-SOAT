FROM golang:1.23-alpine

WORKDIR /app

COPY . .

ENV GOTOOLCHAIN=auto

RUN go mod tidy

RUN go build -o main ./cmd/main.go

CMD ["./main"]