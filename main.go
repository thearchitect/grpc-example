package main

import (
	"context"
	"log"
	"os"
)

//go:generate protoc -I . --go_out=plugins=grpc:./ downloader.proto

func main() {
	log.SetFlags(log.Lmsgprefix | log.Lshortfile | log.Ldate | log.Ltime | log.Lmicroseconds)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(os.Args) > 1 {

		if err := Download(ctx, os.Args[1], os.Args[2], "./tmp"); err != nil {
			panic(err)
		}

	} else {

		RunServer(ctx)

	}
}
