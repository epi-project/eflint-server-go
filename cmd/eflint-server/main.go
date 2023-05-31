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
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&input)

	// Check for parsing errors
	if err != nil {
		log.Println(err)
		output, err := eflint.GenerateJSON(eflint.Output{Success: false})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(output)
		return
	}

	err = eflint.Typecheck(input)

	// Check for typechecking errors
	if err != nil {
		log.Println(err)
		output, err := eflint.GenerateJSON(eflint.Output{Success: false})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(output)
		return
	}

	//pp.Println(input)

	// TODO: Do something with the input
	switch input.Kind {
	case "phrases":
		eflint.InterpretPhrases(input.Phrases)
	case "handshake":
		handshake, err := eflint.GenerateHandshake()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(handshake)
		return
	case "ping":
	default:
		// TODO: This should have been handled by a typecheck function
		http.Error(w, "Unknown kind", http.StatusBadRequest)
		return
	}

	// Write the response
	output, err := eflint.GenerateJSON(eflint.Output{Success: true})
	//output, err := eflint.GenerateJSON(eflint.Output{Success: true, Phrases: input.Phrases})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(output)
	//log.Println("Handled request")

	return
}

func main() {
	http.HandleFunc("/", handler)
	log.Println("Starting at http://localhost:8090")
	log.Fatal(http.ListenAndServe(":8090", nil))
}
