package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/eiannone/keyboard"
)

func executeCmd(cmdName string, args ...string) (*bytes.Buffer, *bytes.Buffer, error) {
	var outb, errb bytes.Buffer
	var cmd *exec.Cmd
	if args != nil {
		cmd = exec.Command(cmdName, args...)
	} else {
		cmd = exec.Command(cmdName)
	}

	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()

	return &outb, &errb, err
}

func printMenu(options []string, selectedIndex int) {
	fmt.Print("\033[H\033[2J") // Clear screen
	fmt.Println("Select file using arrow keys or press ESC to quit:")
	for i, character := range options {
		if i == selectedIndex {
			fmt.Printf("> %s\n", character) // Highlight the selected item
		} else {
			fmt.Printf("  %s\n", character)
		}
	}
}

func main() {
	var option string
	if len(os.Args) > 1 {
		option = os.Args[1]
	} else {
		option = ""
	}

	// Changing the working directory to the ~/.ssh
	homeDir, _ := os.UserHomeDir()
	err := os.Chdir(filepath.Join(homeDir, ".ssh"))
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}

	// Lists all items in ~/.ssh
	outb, errb, err := executeCmd("ls")
	if err != nil {
		fmt.Println("Error executing command:", err)
		fmt.Println("stderr:", errb.String())
		return
	}

	// If the first argument is left blank, or doesn't match one of
	// the items in ~/.ssh, print a menu from which the user can select
	if option == "" || !strings.Contains(outb.String(), option) {
		optionsArray := strings.Split(outb.String(), "\n")
		selectedIndex := 0

		if err := keyboard.Open(); err != nil {
			fmt.Println("Error accessing keyboard:", err)
			panic(err)
		}
		defer keyboard.Close()

		for {
			printMenu(optionsArray, selectedIndex)

			_, key, err := keyboard.GetKey()
			if err != nil {
				fmt.Println("Error getting key:", err)
				panic(err)
			}

			if key == keyboard.KeyArrowUp {
				if selectedIndex > 0 {
					selectedIndex--
				}
			} else if key == keyboard.KeyArrowDown {
				if selectedIndex < len(optionsArray)-1 {
					selectedIndex++
				}
			} else if key == keyboard.KeyEnter {
				option = optionsArray[selectedIndex]
				break
			} else if key == keyboard.KeyEsc {
				fmt.Println("Exiting...")
				return
			}
		}
	}

	outb, errb, err = executeCmd("ssh-agent")
	if err != nil {
		fmt.Println("Error executing command:", err)
		fmt.Println("stderr:", errb.String())
		return
	}

	sshAgentOutput := outb.String()
	lines := strings.Split(sshAgentOutput, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "SSH_AGENT_PID=") || strings.HasPrefix(line, "SSH_AUTH_SOCKET") {
			parts := strings.SplitN(line, ";", 2)
			if len(parts) > 0 {
				env := strings.TrimSpace(parts[0])
				keyValue := strings.SplitN(env, "=", 2)
				if len(keyValue) == 2 {
					os.Setenv(keyValue[0], keyValue[1])
				}
			}
		}
	}

	_, errb, err = executeCmd("ssh-add", option)
	if err != nil {
		fmt.Println("Error executing command:", err)
		fmt.Println("stderr:", errb.String())
		return
	}

	fmt.Println("Identity added:", option)
}
