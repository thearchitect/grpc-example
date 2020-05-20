package main

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path"
)

func main() {
	mux := http.NewServeMux()

	var hashes = map[string]bool{}

	mux.HandleFunc("/chkhash/", func(w http.ResponseWriter, q *http.Request) {
		_, h := path.Split(q.URL.Path)
		if hashes[h] {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, q *http.Request) {
		var blob [1 * 1024 * 1024]byte
		if n, err := rand.Read(blob[:]); err != nil {
			panic(err)
		} else if n != len(blob) {
			panic(errors.New(fmt.Sprintf("size mismatch: %d != %d", n, len(blob))))
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment;filename=useless.blob")
		w.Header().Set("Content-Length", fmt.Sprint(len(blob)))

		h := fmt.Sprintf("%x", sha256.Sum256(blob[:]))

		defer log.Printf("served blob hash: %s", h)

		w.Header().Set("X-Content-Hash", h)

		hashes[h] = true

		if n, err := w.Write(blob[:]); err != nil {
			panic(err)
		} else if n != len(blob) {
			panic(errors.New(fmt.Sprintf("size mismatch: %d != %d", n, len(blob))))
		}
	})

	if err := (&http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: mux,
	}).ListenAndServe(); err != nil {
		panic(err)
	}
}
