package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"atonixcorp/api/internal/database"
	"atonixcorp/api/internal/services"

	"golang.org/x/term"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	fullName, err := readLine(reader, "Super Admin full name: ")
	if err != nil {
		log.Fatalf("read full name: %v", err)
	}
	email, err := readLine(reader, "Super Admin email: ")
	if err != nil {
		log.Fatalf("read email: %v", err)
	}

	password, err := readPassword("Super Admin password (12-72 bytes): ")
	if err != nil {
		log.Fatalf("read password: %v", err)
	}
	confirmation, err := readPassword("Confirm password: ")
	if err != nil {
		log.Fatalf("read password confirmation: %v", err)
	}
	if string(password) != string(confirmation) {
		log.Fatal("password confirmation does not match")
	}

	db := database.Open()
	if err := services.BootstrapSuperAdmin(db, fullName, email, string(password)); err != nil {
		log.Fatalf("bootstrap Super Admin: %v", err)
	}
	fmt.Println("Initial platform Super Admin created. Sign in through the web application.")
}

func readLine(reader *bufio.Reader, prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func readPassword(prompt string) ([]byte, error) {
	fmt.Fprint(os.Stderr, prompt)
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	return password, err
}
