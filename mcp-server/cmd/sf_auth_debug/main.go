package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type sfTokenResponse struct {
	AccessToken string `json:"access_token"`
	InstanceURL string `json:"instance_url"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

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

	loginURL := os.Getenv("SF_LOGIN_URL")
	clientID := os.Getenv("SF_CLIENT_ID")
	clientSecret := os.Getenv("SF_CLIENT_SECRET")
	username := os.Getenv("SF_USERNAME")
	password := os.Getenv("SF_PASSWORD")

	tokenURL := strings.TrimRight(loginURL, "/") + "/services/oauth2/token"

	body := url.Values{}
	body.Set("grant_type", "password")
	body.Set("client_id", clientID)
	body.Set("client_secret", clientSecret)
	body.Set("username", username)
	body.Set("password", password)

	fmt.Printf("Logging in to Salesforce: %s...\n", tokenURL)
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(body.Encode()))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error executing HTTP request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return
	}

	var tok sfTokenResponse
	if err := json.Unmarshal(rawBody, &tok); err != nil {
		fmt.Printf("Error unmarshaling response: %v\n", err)
		return
	}

	if tok.Error != "" {
		fmt.Printf("Salesforce login error: %s - %s\n", tok.Error, tok.ErrorDesc)
		return
	}

	fmt.Println("Login Successful!")
	fmt.Printf("Instance URL: %s\n", tok.InstanceURL)

	// Now query a single account by ID (using the first ID from the list: ACME Corporation 001gK0000181YrMQAU)
	accountID := "001gK0000181YrMQAU"
	fields := "Id,Name,Type,AnnualRevenue,Industry,OwnerId"
	detailURL := fmt.Sprintf("%s/services/data/v59.0/sobjects/Account/%s?fields=%s", strings.TrimRight(tok.InstanceURL, "/"), accountID, fields)

	fmt.Printf("\nQuerying Account detail URL: %s\n", detailURL)
	req, err = http.NewRequest("GET", detailURL, nil)
	if err != nil {
		fmt.Printf("Error creating detail request: %v\n", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("Accept", "application/json")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error executing detail query: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Detail Response Status Code: %d\n", resp.StatusCode)
	rawDetailBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading detail response: %v\n", err)
		return
	}

	fmt.Printf("Raw Detail Response:\n%s\n", string(rawDetailBody))
}
