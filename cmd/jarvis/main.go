package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/symbolichealth/jarvis"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func queryHandler(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON request
	var req struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Validate input
	if strings.TrimSpace(req.Query) == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Query is missing"})
		return
	}

	// Call Jarvis
	j := jarvis.Start()
	resp, err := j.Chat(req.Query)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"response": resp})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/chat", queryHandler)

	// Apply CORS middleware
	handler := corsMiddleware(mux)
	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))

	app := &cli.App{
		Name:  "jarvis",
		Usage: "Interact with the Jarvis assistant",
		Commands: []*cli.Command{
			{
				Name:  "chat",
				Usage: "start an interactive chat session",
				Action: func(c *cli.Context) error {
					if c.NArg() > 0 {
						return cli.Exit("chat takes no arguments", 1)
					}

					j := jarvis.Start()
					reader := bufio.NewReader(os.Stdin)

					for {
						fmt.Print("> ")
						input, err := reader.ReadString('\n')
						if err == io.EOF {
							fmt.Println("\nExiting chat.")
							return nil
						}
						if err != nil {
							return err
						}
						input = strings.TrimSpace(input)
						if input == "" {
							continue
						}

						resp, err := j.Chat(input)
						if err != nil {
							return err
						}
						fmt.Printf("\nJarvis: %s\n", resp)
					}
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
