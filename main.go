package main

// This go:generate directive is a build-time instruction.
// When you run `go generate .` from the project root, it will execute this command.
// This command tells npm to run the 'build' script inside the './ui' directory,
// compiling our Svelte app before the Go program is built.
//go:generate npm --prefix ./ui run build

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
)

//go:embed all:ui/build
var embeddedFiles embed.FS

func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("FATAL: GEMINI_API_KEY environment variable not set")
	}

	db, err := InitDB("./gemini-tool.db")
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize database: %v", err)
	}
	defer db.Close()

	h := NewHandler(db, apiKey)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/prompts", h.PromptsHandler)
	mux.HandleFunc("/api/generate", h.GenerateHandler)
	mux.HandleFunc("/api/chats", h.ChatsHandler)
	mux.HandleFunc("/api/models", h.ModelsHandler)

	uiFS, err := fs.Sub(embeddedFiles, "ui/build")
	if err != nil {
		log.Fatal(err)
	}

	mux.Handle("/", http.FileServer(http.FS(uiFS)))

	loggedMux := LoggingMiddleware(mux)

	log.Println("Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", loggedMux); err != nil {
		log.Fatalf("FATAL: Server failed to start: %v", err)
	}
}
