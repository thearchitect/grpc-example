package main

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
)

func main() {
	if err := (&http.Server{
		Addr: "0.0.0.0:8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
			var blob [1*1024*1024]byte
			if n, err := rand.Read(blob[:]); err != nil {
				panic(err)
			} else if n != len(blob) {
				panic(errors.New(fmt.Sprintf("size mismatch: %d != %d", n, len(blob))))
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", "attachment;filename=useless.blob")
			w.Header().Set("Content-Length", fmt.Sprint(len(blob)))
			w.Header().Set("X-Content-Hash", fmt.Sprintf("%x", sha256.Sum256(blob[:])))
			if n, err := w.Write(blob[:]); err != nil {
				panic(err)
			} else if n != len(blob) {
				panic(errors.New(fmt.Sprintf("size mismatch: %d != %d", n, len(blob))))
			}
		}),
	}).ListenAndServe(); err != nil {
		panic(err)
	}
}
