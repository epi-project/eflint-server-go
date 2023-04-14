package main

import (
	"encoding/json"
	"github.com/Olaf-Erkemeij/eflint-server/internal/eflint"
	"log"
	"net/http"
)

// handler for the root path
func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var input eflint.Input
	err := json.NewDecoder(r.Body).Decode(&input)

	if err != nil {
		output, err := eflint.GenerateJSON(eflint.Output{Success: false})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(output)
		return
	}

	// TODO: Do something with the input
	switch input.Kind {
	case "phrases":
		eflint.InterpretPhrases(input.Phrases)
	case "handshake":
	case "ping":
		break
	default:
		// TODO: This should have been handled by a typecheck function
		http.Error(w, "Unknown kind", http.StatusBadRequest)
		return
	}

	// Write the response
	output, err := eflint.GenerateJSON(eflint.Output{Success: true})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(output)

	return
}

func main() {
	http.HandleFunc("/", handler)
	log.Println("Starting at http://localhost:8090")
	log.Fatal(http.ListenAndServe(":8090", nil))
}
