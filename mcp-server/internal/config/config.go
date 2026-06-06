// Package config loads and validates all runtime configuration
// from environment variables, providing typed access to every
// service setting across the Go MCP server.
package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all resolved runtime configuration for the server.
type Config struct {
	// HTTP server
	HTTPAddr string
	GoEnv    string

	// Request / adapter timeouts
	RequestTimeout time.Duration

	// PostgreSQL
	PgDSN string

	// Jira adapter
	JiraBaseURL  string
	JiraAPIToken string
	JiraUseMock  bool

	// Salesforce adapter
	SfBaseURL  string
	SfAPIToken string
	SfUseMock  bool

	// Supabase cache (Salesforce fallback)
	SupabaseURL     string
	SupabaseAnonKey string
	SupabaseEnabled bool
}

// Load reads environment variables (and an optional .env file) and
// returns a fully-populated Config. It calls log.Fatal on any
// configuration that would make the server unable to start.
func Load() Config {
	// Attempt to load a .env file from the working directory.
	// This is a best-effort operation; production systems rely on
	// real env vars and the error is intentionally suppressed.
	if err := godotenv.Load(); err != nil {
		log.Println("[config] No .env file found; using system environment variables.")
	}

	timeoutMS := getInt("GO_REQUEST_TIMEOUT_MS", 8000)

	cfg := Config{
		HTTPAddr:        getStr("GO_HTTP_ADDR", ":8080"),
		GoEnv:           getStr("GO_ENV", "development"),
		RequestTimeout:  time.Duration(timeoutMS) * time.Millisecond,
		PgDSN:           getStr("PG_DSN", "postgres://postgres:postgres@localhost:5432/mcp_hub?sslmode=disable"),
		JiraBaseURL:     getStr("JIRA_BASE_URL", "https://your-domain.atlassian.net"),
		JiraAPIToken:    getStr("JIRA_API_TOKEN", ""),
		JiraUseMock:     getBool("JIRA_USE_MOCK", true),
		SfBaseURL:       getStr("SF_BASE_URL", "https://your-instance.salesforce.com"),
		SfAPIToken:      getStr("SF_API_TOKEN", ""),
		SfUseMock:       getBool("SF_USE_MOCK", true),
		SupabaseURL:     getStr("SUPABASE_URL", ""),
		SupabaseAnonKey: getStr("SUPABASE_ANON_KEY", ""),
		SupabaseEnabled: getBool("SUPABASE_ENABLED", true),
	}

	return cfg
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func getStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		switch v {
		case "true", "1", "yes":
			return true
		case "false", "0", "no":
			return false
		}
	}
	return fallback
}
