package main

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"

	"google.golang.org/grpc"
)

func Download(ctx context.Context, target, uri, path string) error {
	client, err := NewClient(target)
	if err != nil {
		return err
	}

	name, rc, err := client.DownloadReadCloser(ctx, uri)
	if err != nil {
		return err
	}

	defer rc.Close()

	f, err := os.Create(filepath.Join(path, name))
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err := io.Copy(f, rc); err != nil {
		return err
	}

	log.Println("file downloaded:", name)

	return nil
}

type Client struct {
	c DownloaderServiceClient
}

func NewClient(target string) (*Client, error) {
	c := &Client{}

	if cc, err := grpc.Dial(target, grpc.WithInsecure()); err != nil {
		return nil, err
	} else {
		c.c = NewDownloaderServiceClient(cc)
	}

	return c, nil
}

func (c *Client) DownloadReadCloser(ctx context.Context, uri string) (name string, rc io.ReadCloser, err error) {
	name, wfn, err := c.Download(ctx, uri)
	if err != nil {
		return "", nil, err
	}

	r, w, err := os.Pipe()
	if err != nil {
		return "", nil, err
	}

	go func() {
		defer w.Close()
		if err := wfn(w); err != nil {
			panic(err)
		}
	}()

	return name, r, nil
}

func (c *Client) Download(ctx context.Context, uri string) (name string, wfn func(w io.Writer) error, err error) {
	ctx, cancel := context.WithCancel(ctx)

	defer func() {
		if err != nil {
			cancel()
		}
	}()

	client, err := c.c.DownloadFile(ctx, &DownloadFileReq{URL: uri})
	if err != nil {
		return "", nil, err
	}

	if header, err := client.Recv(); err != nil {
		return "", nil, err
	} else {
		switch header := header.Msg.(type) {
		case *DownloadFileResp_Header:
			name = header.Header.Name
			log.Println("header received:", header.Header.Size, header.Header.ContentType, header.Header.Name)
		default:
			return "", nil, errors.New("bad message type from server")
		}
	}

	return name, func(w io.Writer) error {
		defer cancel()

		for {
			if chunkMsg, err := client.Recv(); err != nil {
				if err == io.EOF {
					log.Println("done")
					break
				} else {
					log.Println("error receiving chunk:", err)
					return err
				}
			} else {
				switch chunk := chunkMsg.Msg.(type) {
				case *DownloadFileResp_Chunk:
					log.Println("chunk received:", len(chunk.Chunk.Chunk))
					if _, err := w.Write(chunk.Chunk.Chunk); err != nil {
						return err
					}
				default:
					return errors.New("bad message type from server")
				}
			}
		}
		return nil
	}, nil
}
