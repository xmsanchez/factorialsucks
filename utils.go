package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"syscall"

	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/ssh/terminal"
)

var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

func readCredentials(c *cli.Context) (string, string) {
	email, exists := os.LookupEnv("FACTORIAL_EMAIL")
	if !exists {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Email: ")
		email, _ = reader.ReadString('\n')
		email = strings.TrimSuffix(email, "\n")
	}
	if !emailRegex.MatchString(email) {
		log.Fatalln("Email not valid")
	}

	var password string
	password, exists = os.LookupEnv("FACTORIAL_PASSWORD")
	if !exists {
		fmt.Print("Password: ")
		bytePassword, _ := terminal.ReadPassword(int(syscall.Stdin))
		password = string(bytePassword)
	}
	if password == "" {
		log.Fatalln("\nNo password provided")
	}
	return email, password
}