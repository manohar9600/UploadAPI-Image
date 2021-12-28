FROM golang:1.18-rc-alpine3.15

WORKDIR /goapp

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY app app
COPY kafka kafka
COPY validators validators
COPY *.go .

RUN go build -o /uploadapi

CMD [ "/uploadapi" ]