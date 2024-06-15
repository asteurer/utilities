package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Credentials struct {
	AccessKeyId     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	SessionToken    string `json:"SessionToken"`
	Expiration      string `json:"Expiration"`
}

type StsResponse struct {
	Credentials Credentials `json:"Credentials"`
}

type AccessKeys struct {
	AccessKeyId       string
	SecretAccessKeyId string
}

func main() {
	// This is the endpoint against which the mfaToken is validated
	serialNumber := os.Getenv("SERIAL_NUMBER")
	mfaToken := os.Getenv("MFA_TOKEN")
	awsAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	awsSecretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	// Remove any existing session token before making the request
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %s\n", err)
	}
	credentialsFilePath := filepath.Join(homeDir, ".aws", "credentials")
	refreshSessionToken(credentialsFilePath, awsAccessKey, awsSecretAccessKey)

	// Run the AWS CLI command
	cmd := exec.Command("aws", "sts", "get-session-token", "--serial-number", serialNumber, "--token-code", mfaToken)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to execute command: %s\nOutput: %s\n", err, string(output))
	}

	// Parse the JSON output
	var response StsResponse
	if err := json.Unmarshal(output, &response); err != nil {
		log.Fatalf("Failed to parse JSON response: %s\n", err)
	}

	// Update the .aws/credentials file
	alterAwsCredentials(credentialsFilePath, response.Credentials)
}

/*
refreshSessionToken replaces the expired credentials with the
IAM user's access and secret access keys.
*/
func refreshSessionToken(credentialsFilePath string, accessKey, secretAccessKey string) {
	updatedContent := fmt.Sprintf("[default]\naws_access_key_id = %s\naws_secret_access_key = %s", accessKey, secretAccessKey)

	// Write the updated content back to the credentials file
	if err := os.WriteFile(credentialsFilePath, []byte(updatedContent), 0644); err != nil {
		log.Fatalf("Failed to write to .aws/credentials file: %s\n", err)
	}
}

func alterAwsCredentials(credentialsFilePath string, credentials Credentials) {
	// Read the existing credentials file
	credentialsContent, err := os.ReadFile(credentialsFilePath)
	if err != nil {
		log.Fatalf("Failed to read .aws/credentials file: %s\n", err)
	}

	// Prepare the new credentials block
	newCredentials := fmt.Sprintf(
		"[default]\naws_access_key_id = %s\naws_secret_access_key = %s\naws_session_token = %s\n",
		credentials.AccessKeyId,
		credentials.SecretAccessKey,
		credentials.SessionToken,
	)

	credentialsString := string(credentialsContent)

	// Check if there is an existing default profile
	if strings.Contains(credentialsString, "[default]") {
		// Replace the existing default profile
		start := strings.Index(string(credentialsString), "[default]")
		end := strings.Index(credentialsString[start:], "\n[")
		if end == -1 {
			end = len(credentialsContent)
		} else {
			end += start
		}
		credentialsContent = []byte(string(credentialsContent[:start]) + newCredentials + string(credentialsContent[end:]))
	} else {
		// Append the new default profile
		credentialsContent = append(credentialsContent, []byte("\n"+newCredentials)...)
	}

	// Write the updated content back to the credentials file
	if err := os.WriteFile(credentialsFilePath, credentialsContent, 0644); err != nil {
		log.Fatalf("Failed to write to .aws/credentials file: %s\n", err)
	}

	fmt.Println("Successfully updated the .aws/credentials file.")
}
