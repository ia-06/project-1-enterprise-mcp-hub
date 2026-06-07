package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Printf("Warning: error loading .env from ../.env: %v. Trying local .env...\n", err)
		err = godotenv.Load(".env")
		if err != nil {
			fmt.Printf("Error: could not load .env file: %v\n", err)
			return
		}
	}

	baseURL := os.Getenv("JIRA_BASE_URL")
	email := os.Getenv("JIRA_EMAIL")
	token := os.Getenv("JIRA_API_TOKEN")

	fmt.Println("--- Jira Environment Configuration ---")
	fmt.Printf("JIRA_BASE_URL:      %s\n", baseURL)
	fmt.Printf("JIRA_EMAIL:         %s\n", email)
	fmt.Printf("JIRA_API_TOKEN:     %s\n", mask(token))
	fmt.Println("--------------------------------------")

	// Call the GET search/jql endpoint with a bounded query and field selections
	projectKey := os.Getenv("JIRA_PROJECT_KEY")
	jql := "project=" + projectKey + " order by updated desc"
	endpoint := fmt.Sprintf("%s/rest/api/3/search/jql?jql=%s&fields=*all&maxResults=5", strings.TrimRight(baseURL, "/"), url.QueryEscape(jql))
	fmt.Printf("\nSending GET request to: %s\n", endpoint)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}
	req.SetBasicAuth(email, token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error executing HTTP request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Response Status Code: %d (%s)\n", resp.StatusCode, resp.Status)
	
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return
	}

	if resp.StatusCode >= 400 {
		fmt.Printf("Raw Response Body:\n%s\n", string(rawBody))
	} else {
		fmt.Printf("Success! Credentials authenticated, and fields were returned successfully.\n")
		fmt.Printf("Full response:\n%s\n", string(rawBody))
		sBody := string(rawBody)
		if len(sBody) > 300 {
			fmt.Printf("First 300 chars of response: %s\n", sBody[:300])
		} else {
			fmt.Printf("Response: %s\n", sBody)
		}
	}
}

func mask(s string) string {
	if len(s) <= 6 {
		return "******"
	}
	return s[:3] + "..." + s[len(s)-3:]
}
