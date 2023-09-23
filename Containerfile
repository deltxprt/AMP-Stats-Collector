# syntax=docker/dockerfile:1

FROM golang:1.20-alpine as build

WORKDIR /app

COPY go.mod ./

RUN go mod download

COPY . .

run go build -o stats main.go

FROM golang:1.20-alpine

WORKDIR /

COPY --from=build /app/stats /stats

CMD [ "/stats" ]

