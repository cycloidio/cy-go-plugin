package main

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/cycloidio/cy-go-plugin/sentry"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

// proxyClient is a shared HTTP client with a 10-second timeout for proxy calls.
var proxyClient = &http.Client{Timeout: 10 * time.Second}

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
	mux.HandleFunc("GET /ui/hello", helloRouter)
	mux.HandleFunc("GET /ui/hello/", helloRouter)

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

// proxyGet calls the main API via PROXY_URL and returns (body, statusCode, error).
func proxyGet(ctx context.Context, apiPath string) ([]byte, int, error) {
	proxyURL := os.Getenv("PROXY_URL")
	secret := os.Getenv("PLUGIN_SECRET")
	if proxyURL == "" {
		return nil, 0, fmt.Errorf("PROXY_URL not set")
	}
	target := buildProxyURL(proxyURL, apiPath, secret)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := proxyClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

// buildProxyURL constructs the proxy target URL, safely handling a PROXY_URL
// that may already contain a query string.
func buildProxyURL(proxyURL, apiPath, secret string) string {
	base := strings.TrimRight(proxyURL, "/")
	path := "/" + strings.TrimLeft(apiPath, "/")
	u, err := url.Parse(base + path)
	if err != nil {
		// Fallback: plain concatenation (shouldn't happen with valid URLs)
		return base + path + "?secret=" + url.QueryEscape(secret)
	}
	q := u.Query()
	if secret != "" {
		q.Set("secret", secret)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func proxyResultHTML(title, org string, body []byte, status int, err error, note string) string {
	safeTitle := html.EscapeString(title)
	safeOrg := html.EscapeString(org)
	noteHTML := ""
	if note != "" {
		noteHTML = "<p><em>" + html.EscapeString(note) + "</em></p>"
	}
	if err != nil {
		return fmt.Sprintf("<h2>%s (org: %s)</h2>%s<p style='color:red'>Error: %s</p>",
			safeTitle, safeOrg, noteHTML, html.EscapeString(err.Error()))
	}
	return fmt.Sprintf("<h2>%s (org: %s) — HTTP %d</h2>%s<pre>%s</pre>",
		safeTitle, safeOrg, status, noteHTML, html.EscapeString(string(body)))
}

func helloRouter(w http.ResponseWriter, r *http.Request) {
	log.Printf("[helloRouter] method=%s path=%s rawQuery=%s", r.Method, r.URL.Path, r.URL.RawQuery)

	subPath := strings.TrimPrefix(r.URL.Path, "/ui/hello")
	subPath = strings.TrimPrefix(subPath, "/")
	log.Printf("[helloRouter] subPath=%q", subPath)

	message := os.Getenv("MESSAGE")
	if message == "" {
		message = "hello world and especially to you <3"
	}
	greetingStyle := os.Getenv("GREETING_STYLE")
	orgCanonical := r.URL.Query().Get("org")

	safeMessage := html.EscapeString(message)
	safeGreetingStyle := html.EscapeString(greetingStyle)
	safeOrg := html.EscapeString(orgCanonical)

	var pageContent string
	switch subPath {
	case "settings":
		pageContent = fmt.Sprintf(`<h1>Settings</h1>
<p><strong>MESSAGE:</strong> %s</p>
<p><strong>GREETING_STYLE:</strong> %s</p>
<p><strong>Organization:</strong> %s</p>`,
			safeMessage, safeGreetingStyle, safeOrg)
	case "about":
		pageContent = `<h1>About</h1>
<p>This is the <strong>cy-go-plugin</strong> demo plugin for Cycloid.</p>
<p>Version: 0.0.9</p>
<p>It demonstrates multi-page navigation inside a plugin iframe widget.</p>`
	case "credentials":
		if orgCanonical == "" {
			pageContent = proxyResultHTML("Credentials", "", nil, 0, fmt.Errorf("org parameter is required"), "")
			break
		}
		body, status, err := proxyGet(r.Context(), "organizations/"+url.PathEscape(orgCanonical)+"/credentials")
		pageContent = proxyResultHTML("Credentials", orgCanonical, body, status, err, "")

	case "projects":
		if orgCanonical == "" {
			pageContent = proxyResultHTML("Projects", "", nil, 0, fmt.Errorf("org parameter is required"), "")
			break
		}
		body, status, err := proxyGet(r.Context(), "organizations/"+url.PathEscape(orgCanonical)+"/projects")
		pageContent = proxyResultHTML("Projects", orgCanonical, body, status, err, "")

	case "roles":
		if orgCanonical == "" {
			pageContent = proxyResultHTML("Roles", "", nil, 0, fmt.Errorf("org parameter is required"), "")
			break
		}
		body, status, err := proxyGet(r.Context(), "organizations/"+url.PathEscape(orgCanonical)+"/roles")
		note := "Note: this plugin does not have organization:role:* scope — a permission error is expected."
		pageContent = proxyResultHTML("Roles", orgCanonical, body, status, err, note)

	default:
		pageContent = fmt.Sprintf(`<h1>Hello World</h1>
<p>%s</p>
<p>Organization: %s</p>`, safeMessage, safeOrg)
		if greetingStyle != "" {
			pageContent += fmt.Sprintf("\n<p>Greeting Style: %s</p>", safeGreetingStyle)
		}
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
<style>
  body { font-family: sans-serif; margin: 0; padding: 16px; }
  nav { background: #f5f5f5; padding: 8px 16px; margin: -16px -16px 16px; display: flex; gap: 16px; }
  nav a { color: #1976d2; text-decoration: none; cursor: pointer; font-weight: 500; }
  nav a:hover { text-decoration: underline; }
</style>
<script>
function navigateTo(subPath) {
  window.parent.postMessage({
    type: 'cycloid:navigate',
    path: subPath
  }, '*');
}
</script>
</head>
<body>
<nav>
  <a onclick="navigateTo(''); return false;" href="#">Home</a>
  <a onclick="navigateTo('settings'); return false;" href="#">Settings</a>
  <a onclick="navigateTo('about'); return false;" href="#">About</a>
  <a onclick="navigateTo('credentials'); return false;" href="#">Credentials</a>
  <a onclick="navigateTo('projects'); return false;" href="#">Projects</a>
  <a onclick="navigateTo('roles'); return false;" href="#">Roles</a>
</nav>
%s
</body>
</html>`, pageContent)
}

func respond(w http.ResponseWriter, request string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"request": request})
}
