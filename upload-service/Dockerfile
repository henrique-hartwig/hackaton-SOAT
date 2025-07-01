FROM golang:1.23-alpine

WORKDIR /app

COPY . .

ENV GOTOOLCHAIN=auto
ENV MINIO_ENDPOINT=minio:9000
ENV MINIO_ACCESS_KEY=minioadmin
ENV MINIO_SECRET_KEY=minioadmin
ENV MINIO_BUCKET=videos

RUN go mod tidy

RUN go build -o main ./cmd/main.go

CMD ["./main"] 