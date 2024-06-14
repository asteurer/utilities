package main

import (
	"bufio"
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
	serialNumber := ""

	var mfaToken string
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter MFA token: ")
		tokenCodeInput, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Failed to read token code: %s\n", err)
		}

		mfaToken = strings.TrimSpace(tokenCodeInput)
		if mfaToken == "" {
			log.Print("mfa token is required")
		} else {
			break
		}
	}

	var awsAccessKey string
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter AWS access key: ")
		accessKeyInput, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Failed to read AWS access key: %s\n", err)
		}

		awsAccessKey = strings.TrimSpace(accessKeyInput)
		if awsAccessKey == "" {
			log.Print("AWS access key is required")
		} else {
			break
		}
	}

	var awsSecretAccessKey string
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter AWS secret access key: ")
		accessKeyInput, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Failed to read AWS secret access key: %s\n", err)
		}

		awsSecretAccessKey = strings.TrimSpace(accessKeyInput)
		if awsSecretAccessKey == "" {
			log.Print("AWS secret access key is required")
		} else {
			break
		}
	}

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
