package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
)

type OnePasswordFields struct {
	Fields []OnePasswordField `json:"fields"`
}

type OnePasswordField struct {
	Label string `json:"label,omitempty"`
	Value string `json:"value,omitempty"`
	OTP   string `json:"totp,omitempty"`
}

type MgmtCredentials struct {
	AccessKeyId     string
	SecretAccessKey string
	MfaToken        string
	SerialNumber    string
}

type StsResponse struct {
	Credentials struct {
		AccessKeyId     string `json:"AccessKeyId"`
		SecretAccessKey string `json:"SecretAccessKey"`
		SessionToken    string `json:"SessionToken"`
		Expiration      string `json:"Expiration"`
	} `json:"Credentials"`
}

type CommandOutput struct {
	StderrBuf bytes.Buffer
	StdoutBuf bytes.Buffer
}

func execCmd(cmd *exec.Cmd, env []string) (CommandOutput, error) {
	var output CommandOutput
	cmd.Stdin = os.Stdin
	cmd.Stdout = &output.StdoutBuf
	cmd.Stderr = &output.StderrBuf

	if env != nil {
		cmd.Env = env
	}

	if err := cmd.Run(); err != nil {
		return output, err
	}

	return output, nil
}

func main() {
	// Get AWS mgmt credentials from 1Password
	cmd := exec.Command("op",
		"item",
		"get",
		"AWS Mgmt Cred",
		"--vault", "Employee",
		"--format", "json",
	)
	opOutput, err := execCmd(cmd, nil)
	if err != nil {
		log.Fatalf("Failed to execute command: %s\nOutput: %s\n", err, opOutput.StderrBuf.String())
	}

	var fields OnePasswordFields
	json.Unmarshal(opOutput.StdoutBuf.Bytes(), &fields)

	var mgmtCreds MgmtCredentials

	for _, object := range fields.Fields {
		switch v := object.Label; v {
		case "one-time password":
			mgmtCreds.MfaToken = object.OTP
		case "AccessKeyId":
			mgmtCreds.AccessKeyId = object.Value
		case "SecretAccessKey":
			mgmtCreds.SecretAccessKey = object.Value
		case "SerialNumber":
			mgmtCreds.SerialNumber = object.Value
		}
	}

	cmd = exec.Command("aws",
		"sts",
		"get-session-token",
		"--serial-number", mgmtCreds.SerialNumber,
		"--token-code", mgmtCreds.MfaToken,
	)
	awsOutput, err := execCmd(cmd, []string{"AWS_ACCESS_KEY_ID=" + mgmtCreds.AccessKeyId, "AWS_SECRET_ACCESS_KEY=" + mgmtCreds.SecretAccessKey})
	if err != nil {
		log.Fatalf("Failed to execute command: %s\nOutput: %s\n", err, awsOutput.StderrBuf.String())
	}

	var response StsResponse
	if err := json.Unmarshal(awsOutput.StdoutBuf.Bytes(), &response); err != nil {
		log.Fatalf("Failed to parse JSON response: %s\n", err)
	}

	cmd = exec.Command("op",
		"item",
		"edit",
		"AWS Temp Cred",
		"--vault", "Employee",
		"AccessKeyId="+response.Credentials.AccessKeyId,
		"SecretAccessKey="+response.Credentials.SecretAccessKey,
		"SessionToken="+response.Credentials.SessionToken,
	)
	_, err = execCmd(cmd, nil)
	if err != nil {
		log.Fatalf("Failed to edit item")
	}

	fmt.Println("AWS credentials have been refreshed")
}
