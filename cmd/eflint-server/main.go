package main

import (
	"io"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Hello, world!")
	})
	log.Println("Starting at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
