package main

import (
	"log"
)

//go:generate protoc -I . --go_out=plugins=grpc:./ downloader.proto

func main() {
	log.Println()
}
