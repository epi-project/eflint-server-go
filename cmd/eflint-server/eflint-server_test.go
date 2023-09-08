package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"github.com/Olaf-Erkemeij/eflint-server/internal/parser"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestServer(t *testing.T) {
	// Go over all the files in the test directory
	// and run the tests

	filepath.WalkDir("tests/correctness", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			t.Fatal(err)
		}

		if d.IsDir() {
			return nil
		}

		t.Run(path, func(t *testing.T) {
			// Open the file
			file, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()

			// Parse the file
			data, err := parser.ParseFile(path, file)

			if err != nil {
				t.Fatal(err)
			}

			// Create a request
			request, _ := http.NewRequest("POST", "/", bytes.NewReader(data))
			response := httptest.NewRecorder()

			// Run the handler
			eFLINTHandler(response, request)

			// Parse the response
			var result map[string]interface{}
			err = json.Unmarshal(response.Body.Bytes(), &result)
			if err != nil {
				t.Fatal(err)
			}

			if result["success"] != true {
				t.Fatal("Expected success to be true")
			}

			results := result["results"].([]interface{})

			file.Seek(0, 0)

			scanner := bufio.NewScanner(file)

			for index := 0; scanner.Scan(); index += 1 {
				res := results[index].(map[string]interface{})

				if queryResult, ok := res["result"]; ok {
					if queryBool, ok := queryResult.(bool); !ok || !queryBool {
						t.Fatal("Query returned false:", scanner.Text())
					}
				}
			}
		})

		return nil
	})
}

func benchmarkDirectoryServer(b *testing.B, path string) {
	filepath.WalkDir(path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			b.Fatal(err)
		}

		if d.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			b.Fatal(err)
		}
		defer file.Close()

		// Parse the file
		data, err := parser.ParseFile(path, file)

		if err != nil {
			b.Fatal(err)
		}

		b.Logf("Sending %v", data)

		b.Run(path, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				// Create a request
				request, _ := http.NewRequest("POST", "/", bytes.NewReader(data))
				response := httptest.NewRecorder()

				// Run the handler
				eFLINTHandler(response, request)
			}
		})

		return nil
	})
}

func benchmarkDirectoryHaskell(b *testing.B, path string) {
	filepath.WalkDir(path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			b.Fatal(err)
		}

		if d.IsDir() {
			return nil
		}

		b.Run(path, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				cmd := exec.Command("/home/olaf/.cabal/bin/eflint-repl", path)
				cmd.Stdin = bytes.NewReader([]byte(""))
				var out bytes.Buffer
				cmd.Stdout = &out

				err := cmd.Run()
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		return nil
	})
}

func BenchmarkServer(b *testing.B) {
	benchmarkDirectoryServer(b, "tests/performance")
}

func BenchmarkHaskell(b *testing.B) {
	benchmarkDirectoryHaskell(b, "tests/performance")
}

func BenchmarkServerDerivation(b *testing.B) {
	benchmarkDirectoryServer(b, "tests/performance/derivation")
}

func BenchmarkHaskellDerivation(b *testing.B) {
	benchmarkDirectoryHaskell(b, "tests/performance/derivation")
}

func BenchmarkServerDimensionality(b *testing.B) {
	benchmarkDirectoryServer(b, "tests/performance/dimensionality")
}

func BenchmarkHaskellDimensionality(b *testing.B) {
	benchmarkDirectoryHaskell(b, "tests/performance/dimensionality")
}

func BenchmarkServerCombinatorial(b *testing.B) {
	benchmarkDirectoryServer(b, "tests/performance/combinatorial")
}
