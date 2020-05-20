package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"testing"
)

//func TestMain(m *testing.M) {
//	m.Run()
//}

func TestDownload(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go RunServer(ctx)

	client, err := NewClient("localhost:7532")
	if err != nil {
		t.Fatal(err)
	}

	name, writeTo, err := client.Download(ctx, "http://test_server:8080/")
	if err != nil {
		t.Fatal(err)
	}

	if name != "useless.blob" {
		t.Fatalf("wrong file name: %s", name)
	}

	buf := bytes.NewBuffer([]byte{})

	if err := writeTo(buf); err != nil {
		t.Fatal(err)
	}

	h := fmt.Sprintf("%x", sha256.Sum256(buf.Bytes()))

	t.Logf("downloaded blob hash: %s", h)

	if resp, err := http.Get(fmt.Sprintf("http://test_server:8080/chkhash/%s", h)); err != nil {
		t.Error(err)
		t.FailNow()
	} else {
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("bad response: %d", resp.StatusCode)
		}
	}

}
