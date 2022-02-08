FROM golang:1.18-rc-buster

RUN apt-get install -y ca-certificates

WORKDIR /goapp

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY app app
COPY kafka kafka
COPY validators validators
COPY *.go .

RUN go build -o /uploadapi

EXPOSE 8002
CMD [ "/uploadapi" ]