FROM golang:latest AS builder
WORKDIR /app
COPY . .
RUN go mod init boxes
RUN go get golang.org/x/crypto/bcrypt
RUN go get github.com/AlexanderGrom/componenta/crypt
RUN go get github.com/go-sql-driver/mysql
RUN go get github.com/go-telegram-bot-api/telegram-bot-api/v5
RUN go mod download

ENV CGO_ENABLED=0
ENV GOOS=linux

RUN go build -o main .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/help.txt .
CMD ["./main"]

ENV TELEGRAM_APITOKEN=<token>
ENV MYSQL_HOST=mysql
ENV MYSQL_USER=root
ENV MYSQL_DB=boxes
