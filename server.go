package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RunServer(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// FIXME: How to setup base context for gRPC server in a more native way?
	server := grpc.NewServer(
		grpc_middleware.WithUnaryServerChain(func(ctx2 context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
			ctx2, cancel := context.WithCancel(ctx2)
			defer cancel()

			go func() {
				defer cancel()
				select {
				case <-ctx.Done():
				case <-ctx2.Done():
				}
			}()

			return handler(ctx2, req)
		}),
		grpc_middleware.WithStreamServerChain(func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			ctx2, cancel := context.WithCancel(ss.Context())
			defer cancel()

			go func() {
				defer cancel()
				select {
				case <-ctx.Done():
				case <-ctx2.Done():
				}
			}()

			wrapped := grpc_middleware.WrapServerStream(ss)
			wrapped.WrappedContext = ctx2

			return handler(srv, wrapped)
		}),
	)

	RegisterDownloaderServiceServer(server, &downloaderService{})

	go func() {
		<-ctx.Done()
		server.GracefulStop()
	}()

	if lis, err := net.Listen("tcp4", "0.0.0.0:7532"); err != nil {
		panic(err)
	} else if err := server.Serve(lis); err != nil {
		panic(err)
	}
}

type downloaderService struct{}

func (ds *downloaderService) DownloadFile(req *DownloadFileReq, server DownloaderService_DownloadFileServer) error {
	client := &http.Client{}

	q, err := http.NewRequestWithContext(server.Context(), "GET", req.URL, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(q)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg := func() string {
			return fmt.Sprintf("%d - %s", resp.StatusCode, resp.Status)
		}

		switch resp.StatusCode {
		case http.StatusNotFound:
			return status.Error(codes.NotFound, msg())
		default:
			return status.Error(codes.Internal, msg())
		}
	}

	{ // HEADER
		var (
			header DownloadFileHeader
			path   string
		)

		if u, err := url.Parse(req.URL); err != nil {
			return status.Error(codes.InvalidArgument, fmt.Sprintf("malformed url: %s", err.Error()))
		} else {
			path = u.Path
		}

		if ct := resp.Header.Get("Content-Type"); ct != "" {
			header.ContentType = ct
		}

		if cd := resp.Header.Get("Content-Disposition"); cd != "" {
			_, params, err := mime.ParseMediaType(cd)
			if err != nil {
				log.Println("error parsing content-disposition header:", err)
			} else if filename := params["filename"]; filename != "" {
				header.Name = filename
			}
		}
		if header.Name == "" {
			i := strings.LastIndex(path, "/")
			if i > 0 && i < len(path)-1 {
				header.Name = path[i+1:]
			}
		}
		if header.Name == "" {
			header.Name = fmt.Sprintf("%x", sha256.Sum256([]byte(path)))
		}
		if filepath.Ext(header.Name) == "" {
			if exts, err := mime.ExtensionsByType(header.ContentType); err == nil && len(exts) > 0 {
				header.Name += exts[0]
			}
		}

		if cl := resp.Header.Get("Content-Length"); cl != "" {
			if size, err := strconv.Atoi(cl); err != nil {
				log.Println("error parsing content-length:", err)
			} else {
				header.Size = int64(size)
			}
		}

		if err := server.Send(&DownloadFileResp{
			Msg: &DownloadFileResp_Header{
				Header: &header,
			},
		}); err != nil {
			return status.Error(codes.Internal, err.Error())
		} else {
			log.Println("header sent:", header.Size, header.ContentType, header.Name)
		}
	}

	var buf [32 * 1024]byte
	for { // CHUNKS
		n, err := resp.Body.Read(buf[:])
		if err != nil && err != io.EOF {
			return status.Error(codes.Internal, err.Error())
		} else {
			log.Println("chunk read:", n)
			chunkData := buf[:n]

			if sendErr := server.Send(&DownloadFileResp{
				Msg: &DownloadFileResp_Chunk{
					Chunk: &DownloadFileChunk{
						Chunk: chunkData,
					},
				},
			}); sendErr != nil {
				return status.Error(codes.Internal, sendErr.Error())
			} else {
				log.Println("chunk sent:", len(chunkData))
			}

			if err == io.EOF {
				log.Println("done")
				return nil
			}
		}
	}
}
