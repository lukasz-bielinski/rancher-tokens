package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"rancher-tokens/vaultlogic"
	"time"

	"github.com/tidwall/gjson"
)

func logJSON(message string) {
	logMessage := map[string]string{"message": message}
	messageJSON, err := json.Marshal(logMessage)
	if err != nil {
		log.Fatalf("Could not marshal log message: %v", err)
	}
	log.Println(string(messageJSON))
}

func main() {
	RANCHER_SERVER := os.Getenv("RANCHER_SERVER")
	USERNAME := os.Getenv("USERNAME")
	PASSWORD := os.Getenv("PASSWORD")

	currentTime := time.Now().UTC().Format(time.RFC3339)
	API_KEY_DESCRIPTION := fmt.Sprintf("Token created at %s with cronjob rancher-tokens", currentTime)

	if RANCHER_SERVER == "" || USERNAME == "" || PASSWORD == "" {
		logJSON("Environment variables are not set.")
		return
	}

	var client *http.Client
	if os.Getenv("SKIP_TLS_VERIFY") == "true" {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{Transport: tr}
	} else {
		client = &http.Client{}
	}

	for {
		resp, err := client.Get(fmt.Sprintf("https://%s", RANCHER_SERVER))
		if err == nil && resp.StatusCode == http.StatusOK {
			logJSON("Endpoint is accessible.")
			break
		} else {
			logJSON("Waiting for endpoint to be accessible...")
			time.Sleep(5 * time.Second)
		}
	}

	// Login and Create API Key sections need to use the custom HTTP client too.

	// For Login
	req, err := http.NewRequest("POST", fmt.Sprintf("https://%s/v3-public/localProviders/local?action=login", RANCHER_SERVER), bytes.NewBuffer([]byte(fmt.Sprintf(`{"username":"%s","password":"%s"}`, USERNAME, PASSWORD))))
	if err != nil {
		logJSON(fmt.Sprintf("Login request failed: %s", err))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		logJSON(fmt.Sprintf("Login request failed: %s", err))
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logJSON(fmt.Sprintf("Could not read response: %s", err))
		return
	}

	LOGIN_TOKEN := gjson.Get(string(body), "token").String()
	if LOGIN_TOKEN == "" {
		logJSON("Login failed.")
		return
	}

	// Create API Key
	req, err = http.NewRequest(
		"POST",
		fmt.Sprintf("https://%s/v3/tokens", RANCHER_SERVER),
		bytes.NewBuffer([]byte(fmt.Sprintf(`{"type":"token", "description":"%s"}`, API_KEY_DESCRIPTION))),
	)
	if err != nil {
		logJSON(fmt.Sprintf("API key request creation failed: %s", err))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", LOGIN_TOKEN))

	resp, err = client.Do(req)
	if err != nil {
		logJSON(fmt.Sprintf("API key request failed: %s", err))
		return
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		logJSON(fmt.Sprintf("Could not read API key response: %s", err))
		return
	}

	API_KEY_NAME := gjson.Get(string(body), "name").String()
	API_KEY_TOKEN := gjson.Get(string(body), "token").String()

	if API_KEY_NAME == "" || API_KEY_TOKEN == "" {
		logJSON(fmt.Sprintf("API key creation failed. Status Code: %d, Response: %s", resp.StatusCode, string(body)))
		return
	}

	data := map[string]interface{}{
		"rancher2_access_key": API_KEY_NAME,
		"rancher2_secret_key": API_KEY_TOKEN,
	}

	_, err = vaultlogic.GetSecretWithKubernetesAuth(data)
	if err != nil {
		logJSON(fmt.Sprintf("Failed to get or set secret: %s", err))
		return
	}

}
