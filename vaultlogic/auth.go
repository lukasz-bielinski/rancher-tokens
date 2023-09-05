package vaultlogic

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/kubernetes"
)

// LogJSON logs a JSON-formatted message
func logJSON(message string) {
	logMessage := map[string]string{"message": message}
	messageJSON, err := json.Marshal(logMessage)
	if err != nil {
		log.Fatalf("Could not marshal log message: %v", err)
	}
	log.Println(string(messageJSON))
}

// LogAndReturnError logs an error message and returns an error
func logAndReturnError(errMessage string, originalErr error) error {
	if originalErr != nil {
		errMessage = fmt.Sprintf("%s: %s", errMessage, originalErr.Error())
	}
	logJSON(errMessage)
	return fmt.Errorf(errMessage)
}

// GetSecretWithKubernetesAuth gets or updates a secret in Vault using Kubernetes authentication
func GetSecretWithKubernetesAuth(dataToStore map[string]interface{}) (string, error) {
	VAULT_ADDR := os.Getenv("VAULT_ADDR")
	if len(VAULT_ADDR) == 0 {
		return "", logAndReturnError("VAULT_ADDR not set", nil)
	}

	secretEngine := os.Getenv("VAULT_SECRET_ENGINE")
	if secretEngine == "" {
		secretEngine = "kv-v2"
	}

	secretPath := os.Getenv("VAULT_SECRET_PATH")
	if secretPath == "" {
		secretPath = "creds"
	}

	config := &vault.Config{
		Address: "http://" + VAULT_ADDR + ":8200",
	}

	client, err := vault.NewClient(config)
	if err != nil {
		return "", logAndReturnError("Unable to initialize Vault client", err)
	}

	k8sAuth, err := auth.NewKubernetesAuth(
		"rancher-tokens",
		auth.WithServiceAccountTokenPath("/var/run/secrets/kubernetes.io/serviceaccount/token"),
	)
	if err != nil {
		return "", logAndReturnError("Unable to initialize Kubernetes auth method", err)
	}

	authInfo, err := client.Auth().Login(context.TODO(), k8sAuth)
	if err != nil {
		return "", logAndReturnError("Unable to log in with Kubernetes auth", err)
	}
	if authInfo == nil {
		return "", logAndReturnError("No auth info was returned after login", nil)
	}

	kv := client.KVv2(secretEngine)
	secret, err := kv.Get(context.Background(), secretPath)

	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			// Create the secret if it doesn't exist
			_, err = kv.Put(context.Background(), "creds", dataToStore)
			if err != nil {
				return "", logAndReturnError("Unable to create secret", err)
			}
			logJSON("Successfully created secret")
			return dataToStore["password"].(string), nil
		}
		return "", logAndReturnError("Unable to read secret", err)
	}

	// Update the secret if it already exists
	_, err = kv.Put(context.Background(), "creds", dataToStore)
	if err != nil {
		return "", logAndReturnError("Unable to update secret", err)
	}
	logJSON("Successfully updated existing secret")

	value, ok := secret.Data["rancher2_access_key"].(string)
	if !ok {
		return "", logAndReturnError("Value type assertion failed", nil)
	}

	logJSON("Successfully fetched secret")
	return value, nil
}
