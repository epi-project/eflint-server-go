package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Olaf-Erkemeij/eflint-server/internal/eflint"
)

// handler for the root path
func eFLINTHandler(w http.ResponseWriter, r *http.Request) {
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
		log.Println("Handling phrases request")

		log.Println("Interpreting phrases...")
		eflint.InterpretPhrases(input.Phrases)

		// Write the response
		log.Println("Generating JSON...")
		output, err := eflint.GenerateJSON(eflint.Output{Success: true})
		//output, err := eflint.GenerateJSON(eflint.Output{Success: true, Phrases: input.Phrases})

		log.Println("Considering errors...")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Println("Writing output...")
		w.Write(output)

		log.Println("Handled phrases request")
		return

	case "handshake":
		handshake, err := eflint.GenerateHandshake()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(handshake)
		return
	case "ping":
		return
	default:
		// TODO: This should have been handled by a typecheck function
		http.Error(w, "Unknown kind", http.StatusBadRequest)
		return
	}
}

func main() {
	http.HandleFunc("/", eFLINTHandler)
	log.Println("Starting at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
