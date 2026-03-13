FROM golang:1.25.5-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o api ./cmd/api
RUN go build -o worker ./cmd/worker

RUN apk add --no-cache docker-cli

CMD [ "./api" ]
