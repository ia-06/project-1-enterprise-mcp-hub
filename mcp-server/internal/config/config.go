// Package config loads and validates all runtime configuration
// from environment variables (and an optional .env file), providing
// typed, zero-ambiguity access to every service credential and tuning
// parameter across the Go MCP + JSON-RPC server.
package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all resolved runtime configuration for the server.
// Fields are grouped by service to match the .env layout.
type Config struct {
	// ---- HTTP server --------------------------------------------------------
	HTTPAddr string
	GoEnv    string

	// ---- Request / adapter timeouts -----------------------------------------
	// All outbound HTTP adapter calls (Jira, Salesforce, Supabase) share this
	// deadline. When Salesforce exceeds it, the Supabase fallback triggers.
	RequestTimeout time.Duration

	// ---- PostgreSQL ---------------------------------------------------------
	PgDSN string

	// ---- Jira Cloud — REST API v3 (Basic Auth) ------------------------------
	// Authentication: base64(JiraEmail + ":" + JiraAPIToken) in Authorization header.
	JiraBaseURL        string
	JiraEmail          string
	JiraAPIToken       string
	JiraProjectKey     string
	JiraSFAccountField string // JQL field name mapping to Salesforce Account ID
	JiraUseMock        bool

	// ---- Salesforce REST API — OAuth2 ROPC flow -----------------------------
	// The adapter exchanges (ClientID, ClientSecret, Username, Password) for a
	// short-lived bearer token at SF_LOGIN_URL/services/oauth2/token.
	SfBaseURL      string
	SfLoginURL     string
	SfClientID     string
	SfClientSecret string
	SfUsername     string
	SfPassword     string // concatenation of password + security token
	SfAPIVersion   string
	SfUseMock      bool

	// ---- Supabase cache (Salesforce Scenario-A fallback) --------------------
	SupabaseURL            string
	SupabaseAnonKey        string
	SupabaseServiceRoleKey string
	SupabaseEnabled        bool
}

// Load reads environment variables (and an optional root .env file) and
// returns a fully-populated Config. Warnings are logged for missing
// live credentials when mock mode is disabled; the server still starts
// so that health routes remain reachable.
func Load() Config {
	// Best-effort .env load — production systems supply real env vars directly.
	if err := godotenv.Load(); err != nil {
		log.Println("[config] No .env file found; using system environment variables.")
	}

	timeoutMS := getInt("GO_REQUEST_TIMEOUT_MS", 8000)

	cfg := Config{
		// HTTP
		HTTPAddr:       getStr("GO_HTTP_ADDR", ":8080"),
		GoEnv:          getStr("GO_ENV", "development"),
		RequestTimeout: time.Duration(timeoutMS) * time.Millisecond,

		// Postgres
		PgDSN: getStr("PG_DSN", "postgres://postgres:postgres@localhost:5432/mcp_hub?sslmode=disable"),

		// Jira
		JiraBaseURL:        getStr("JIRA_BASE_URL", ""),
		JiraEmail:          getStr("JIRA_EMAIL", ""),
		JiraAPIToken:       getStr("JIRA_API_TOKEN", ""),
		JiraProjectKey:     getStr("JIRA_PROJECT_KEY", ""),
		JiraSFAccountField: getStr("JIRA_SF_ACCOUNT_FIELD", "cf[Salesforce Account ID]"),
		JiraUseMock:        getBool("JIRA_USE_MOCK", true),

		// Salesforce
		SfBaseURL:      getStr("SF_BASE_URL", ""),
		SfLoginURL:     getStr("SF_LOGIN_URL", "https://login.salesforce.com"),
		SfClientID:     getStr("SF_CLIENT_ID", ""),
		SfClientSecret: getStr("SF_CLIENT_SECRET", ""),
		SfUsername:     getStr("SF_USERNAME", ""),
		SfPassword:     getStr("SF_PASSWORD", ""),
		SfAPIVersion:   getStr("SF_API_VERSION", "v59.0"),
		SfUseMock:      getBool("SF_USE_MOCK", true),

		// Supabase
		SupabaseURL:            getStr("SUPABASE_URL", ""),
		SupabaseAnonKey:        getStr("SUPABASE_ANON_KEY", ""),
		SupabaseServiceRoleKey: getStr("SUPABASE_SERVICE_ROLE_KEY", ""),
		SupabaseEnabled:        getBool("SUPABASE_ENABLED", false),
	}

	cfg.validate()
	return cfg
}

// validate emits warnings when live-mode credentials are missing.
// It never calls log.Fatal — the server starts in a degraded state so
// health checks and non-affected routes keep working.
func (c Config) validate() {
	if !c.JiraUseMock {
		if c.JiraBaseURL == "" || c.JiraEmail == "" || c.JiraAPIToken == "" {
			log.Println("[config] WARNING: JIRA_USE_MOCK=false but JIRA_BASE_URL / JIRA_EMAIL / JIRA_API_TOKEN are not fully set.")
		}
	}
	if !c.SfUseMock {
		if c.SfBaseURL == "" || c.SfClientID == "" || c.SfClientSecret == "" ||
			c.SfUsername == "" || c.SfPassword == "" {
			log.Println("[config] WARNING: SF_USE_MOCK=false but Salesforce OAuth credentials are not fully set.")
		}
	}
	if c.SupabaseEnabled && (c.SupabaseURL == "" || c.SupabaseAnonKey == "") {
		log.Println("[config] WARNING: SUPABASE_ENABLED=true but SUPABASE_URL / SUPABASE_ANON_KEY are not set. Cache fallback is disabled.")
	}
}

// ---------------------------------------------------------------------------
// Private helpers
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
