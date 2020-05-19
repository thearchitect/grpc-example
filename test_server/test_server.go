package main

import "net/http"

func main() {
	if err := (&http.Server{
		Addr: "0.0.0.0:8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {


		}),
	}).ListenAndServe(); err != nil {
		panic(err)
	}
}
