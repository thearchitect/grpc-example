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
#### pkgs container (to speed up rebuild after context change; 2x in this case)
FROM base as pkgs

WORKDIR /src

#RUN printf "\
#        package main\n\n\
#        import (\n\
#            _ \"github.com/golang/protobuf/proto\"\n\
#            _ \"github.com/grpc-ecosystem/go-grpc-middleware\"\n\
#            _ \"google.golang.org/grpc\"\n\
#        )\n\n\
#        func main() {}\n" > ./stub.go
#RUN go build

COPY ./go.mod ./go.sum ./

RUN go mod download -x

################################################################
#### final container
FROM base

COPY --from=pkgs $GOPATH/ $GOPATH/

WORKDIR /src
COPY . .

RUN go generate
RUN go build -o /grpc-example

WORKDIR /

ENV DOCKER true
