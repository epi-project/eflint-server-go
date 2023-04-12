package main

import (
	"github.com/Olaf-Erkemeij/eflint-server/internal/eflint"
	"log"
	"net/http"
)

// handler for the root path
func handler(w http.ResponseWriter, r *http.Request) {
	// Try to parse the request body as JSON into the Input struct
	// If it fails, return a 400 Bad Request
	// If it succeeds, return a 200 OK
	input, err := eflint.ParseInput(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: Do something with the input
	switch input.Kind {
	case "phrases":
	case "handshake":
	case "ping":
		break
	default:
		http.Error(w, "Unknown kind", http.StatusBadRequest)
		return
	}

	log.Println("Received request:", input)

	// Write the response
	output, err := eflint.GenerateJSON(eflint.Output{true, []interface{}{input.Phrases}})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(output)

	return
}

func main() {
	http.HandleFunc("/", handler)
	log.Println("Starting at http://localhost:8090")
	log.Fatal(http.ListenAndServe(":8090", nil))
}
