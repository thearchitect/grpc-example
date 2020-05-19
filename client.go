package main

import (
	"context"
	"errors"
	"google.golang.org/grpc"
	"io"
	"log"
	"os"
	"path/filepath"
)

func Download(ctx context.Context, target, uri, path string) error {
	client, err := NewClient(target)
	if err != nil {
		return err
	}

	name, rc, err := client.Download(ctx, uri)
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

func (c *Client) Download(ctx context.Context, uri string) (name string, rc io.ReadCloser, err error) {
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

	proxy := newReadCloser(ctx, func() error {
		cancel()
		return nil
	})

	go func() {
		defer cancel()

		for {
			if chunk, err := client.Recv(); err != nil {
				if err == io.EOF {
					log.Println("done")
					go proxy.Feed(nil)
				} else {
					log.Println("error receiving chunk:", err)
				}
				return
			} else {
				switch chunk := chunk.Msg.(type) {
				case *DownloadFileResp_Chunk:
					log.Println("chunk received:", len(chunk.Chunk.Chunk))
					go proxy.Feed(chunk.Chunk.Chunk)
				default:
					panic(errors.New("bad message type from server"))
				}
			}
		}
	}()

	return name, proxy, nil
}

var _ io.ReadCloser = new(readCloser)

type readCloser struct {
	ctx    context.Context
	reader chan []byte
	close  func() error

	buf []byte
}

func newReadCloser(ctx context.Context, close func() error) *readCloser {
	rc := &readCloser{
		ctx:    ctx,
		close:  close,
		reader: make(chan []byte),
	}

	return rc
}

func (r *readCloser) Feed(p []byte) {
	r.reader <- p
}

func (r *readCloser) Read(p []byte) (n int, err error) {
	flush := func() (n int, err error) {
		n = copy(p, r.buf)
		r.buf = r.buf[n:]
		return n, nil
	}
	if len(r.buf) > 0 {
		return flush()
	}

	select {
	case data, ok := <-r.reader:
		if ok && len(data) > 0 {
			r.buf = data
			return flush()
		} else {
			log.Println("done: eof")
			return 0, io.EOF
		}
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	}
}

func (r *readCloser) Close() error {
	return r.close()
}
