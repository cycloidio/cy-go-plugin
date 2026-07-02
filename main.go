package main

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/cycloidio/cy-go-plugin/sentry"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

func main() {
	dbFile := os.Getenv("DB_FILE")

	dsn := ":memory:"
	if dbFile != "" {
		dsn = dbFile
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if dbFile == "" {
		db.SetMaxOpenConns(1)
	}

	if _, err := db.Exec(schema); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := sentry.Seed(db); err != nil {
		log.Fatalf("seed: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /_cy/ping", ping)
	mux.HandleFunc("GET /_cy/test-proxy", testProxy)
	mux.HandleFunc("POST /_cy/events", events)
	mux.HandleFunc("DELETE /_cy/plugin", func(w http.ResponseWriter, r *http.Request) {
		if err := sentry.Clear(db); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		respond(w, "plugin")
	})
	mux.HandleFunc("POST /_cy/resync", func(w http.ResponseWriter, r *http.Request) {
		if err := sentry.Clear(db); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := sentry.Seed(db); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		respond(w, "resync")
	})
	mux.HandleFunc("GET /sentry/iframe", sentry.IframeHandler)
	mux.HandleFunc("GET /ui/hello", hello)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server is running on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func ping(w http.ResponseWriter, _ *http.Request) {
	respond(w, "ping")
}

// testProxy calls the plugin manager's internal proxy endpoint using the injected
// PROXY_URL and PLUGIN_SECRET env vars, and forwards the response back to the caller.
func testProxy(w http.ResponseWriter, r *http.Request) {
	proxyURL := os.Getenv("PROXY_URL")
	if proxyURL == "" {
		http.Error(w, "PROXY_URL not set", http.StatusServiceUnavailable)
		return
	}

	targetURL := proxyURL
	if secret := os.Getenv("PLUGIN_SECRET"); secret != "" {
		targetURL += "?secret=" + url.QueryEscape(secret)
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("build request: %v", err), http.StatusInternalServerError)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("proxy call failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func events(w http.ResponseWriter, _ *http.Request) {
	respond(w, "events")
}

func hello(w http.ResponseWriter, r *http.Request) {
	message := os.Getenv("MESSAGE")
	if message == "" {
		message = "hello world and especially to you <3"
	}
	greetingStyle := os.Getenv("GREETING_STYLE")
	orgCanonical := r.URL.Query().Get("org")
	w.Header().Set("Content-Type", "text/html")
	html := fmt.Sprintf("<h1>Hello World</h1>\n<p>%s</p>\n<p>Organization: %s</p>", message, orgCanonical)
	if greetingStyle != "" {
		html += fmt.Sprintf("\n<p>Greeting Style: %s</p>", greetingStyle)
	}
	fmt.Fprint(w, html)
}

func respond(w http.ResponseWriter, request string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"request": request})
}
