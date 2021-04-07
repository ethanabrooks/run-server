FROM golang:latest
WORKDIR /go/src/app
COPY go.mod .

RUN go get -d -v ./...
RUN go install -v ./...

COPY . .