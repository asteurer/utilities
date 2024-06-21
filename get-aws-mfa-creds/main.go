package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
	serialNumber := os.Getenv("SERIAL_NUMBER") // ARN of the MFA device
	mfaToken := os.Getenv("MFA_TOKEN")
	awsAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	awsSecretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	// Remove any existing session token before making the request
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %s\n", err)
	}
	credentialsFilePath := filepath.Join(homeDir, ".aws", "credentials")

	cmd := exec.Command("aws", "sts", "get-session-token", "--serial-number", serialNumber, "--token-code", mfaToken)
	cmd.Env = []string{"AWS_ACCESS_KEY_ID=" + awsAccessKey, "AWS_SECRET_ACCESS_KEY=" + awsSecretAccessKey}
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to execute command: %s\nOutput: %s\n", err, string(output))
	}

	var response StsResponse
	if err := json.Unmarshal(output, &response); err != nil {
		log.Fatalf("Failed to parse JSON response: %s\n", err)
	}

	placeCredentials(credentialsFilePath, response.Credentials)
}

// placeCredentials places the temporary credentials generated by the `aws sts` command into ~/.aws/credentials
func placeCredentials(credentialsFilePath string, credentials Credentials) {
	newCredentials := "[default]" +
		"\naws_access_key_id = " + credentials.AccessKeyId +
		"\naws_secret_access_key = " + credentials.SecretAccessKey +
		"\naws_session_token = " + credentials.SessionToken

	if err := os.WriteFile(credentialsFilePath, []byte(newCredentials), 0644); err != nil {
		log.Fatalf("Failed to write to .aws/credentials file: %s\n", err)
	}

	fmt.Println("Successfully updated the .aws/credentials file with temporary credentials.")
}
