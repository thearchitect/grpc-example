################################################################
#### base image
FROM golang:1.14-buster as base

RUN apt-get update && apt-get -y install git unzip build-essential autoconf libtool

RUN git clone https://github.com/google/protobuf.git /protobuf
RUN cd /protobuf &&\
    ./autogen.sh &&\
    ./configure &&\
    make &&\
    make install &&\
    ldconfig &&\
    make clean &&\
    cd .. &&\
    rm -r protobuf

RUN go get google.golang.org/grpc
RUN go get github.com/golang/protobuf/protoc-gen-go

################################################################
#### final container
FROM base

WORKDIR /src
COPY . .

RUN go generate
RUN go build -o /grpc-example

WORKDIR /

ENV DOCKER true
