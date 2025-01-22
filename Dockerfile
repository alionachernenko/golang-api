FROM golang:1.22

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

WORKDIR /app/cmd/app

RUN go build -o /app/auth-service .

CMD [ "/app/auth-service" ]