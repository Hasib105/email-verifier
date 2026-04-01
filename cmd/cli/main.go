package main

import (
	"bufio"
	"context"
	"email-verifier-api/internal/config"
	"email-verifier-api/internal/repo"
	"email-verifier-api/internal/service"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "signup":
		runSignup()
	case "createsuperuser":
		runCreateSuperuser()
	case "list-users":
		runListUsers()
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Email Verifier CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  cli signup           - Create a new user account")
	fmt.Println("  cli createsuperuser  - Create a superuser account (admin)")
	fmt.Println("  cli list-users       - List all users")
	fmt.Println("  cli help             - Show this help message")
	fmt.Println()
}

func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // newline after password input
	if err != nil {
		return "", err
	}
	return string(bytePassword), nil
}

func runCreateSuperuser() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== Email Verifier - Create Superuser ===")
	fmt.Println()

	// Get name
	fmt.Print("Name: ")
	name, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading input: %v\n", err)
		os.Exit(1)
	}
	name = strings.TrimSpace(name)

	if name == "" {
		fmt.Println("Error: Name is required")
		os.Exit(1)
	}

	// Get email
	fmt.Print("Email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading input: %v\n", err)
		os.Exit(1)
	}
	email = strings.TrimSpace(email)

	if email == "" {
		fmt.Println("Error: Email is required")
		os.Exit(1)
	}

	// Get password
	password, err := readPassword("Password: ")
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}

	if len(password) < 6 {
		fmt.Println("Error: Password must be at least 6 characters")
		os.Exit(1)
	}

	// Confirm password
	confirmPassword, err := readPassword("Password (again): ")
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}

	if password != confirmPassword {
		fmt.Println("Error: Passwords do not match")
		os.Exit(1)
	}

	// Connect to database
	cfg := config.Load()
	repository, err := repo.New(cfg.ResolveDatabaseDSN())
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		fmt.Println("Make sure the database is running and configured correctly.")
		os.Exit(1)
	}
	defer repository.Close()

	userService := service.NewUserService(repository)

	// Create superuser
	result, err := userService.CreateSuperuser(context.Background(), service.CreateSuperuserRequest{
		Name:     name,
		Email:    email,
		Password: password,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Superuser created successfully!")
	fmt.Println()
	fmt.Printf("  User ID:  %s\n", result.User.ID)
	fmt.Printf("  Email:    %s\n", result.User.Email)
	fmt.Printf("  API Key:  %s\n", result.APIKey)
	fmt.Println()
}

func runSignup() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== Email Verifier - User Signup ===")
	fmt.Println()

	// Get name
	fmt.Print("Enter your name: ")
	name, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading input: %v\n", err)
		os.Exit(1)
	}
	name = strings.TrimSpace(name)

	if name == "" {
		fmt.Println("Name is required")
		os.Exit(1)
	}

	// Get email
	fmt.Print("Enter your email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading input: %v\n", err)
		os.Exit(1)
	}
	email = strings.TrimSpace(email)

	if email == "" {
		fmt.Println("Email is required")
		os.Exit(1)
	}

	password, err := readPassword("Enter password: ")
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}
	if len(password) < 6 {
		fmt.Println("Password must be at least 6 characters")
		os.Exit(1)
	}

	confirmPassword, err := readPassword("Confirm password: ")
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}
	if password != confirmPassword {
		fmt.Println("Passwords do not match")
		os.Exit(1)
	}

	// Connect to database
	cfg := config.Load()
	repository, err := repo.New(cfg.ResolveDatabaseDSN())
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		fmt.Println("Make sure the database is running and configured correctly.")
		os.Exit(1)
	}
	defer repository.Close()

	userService := service.NewUserService(repository)

	// Create user
	req := service.SignupRequest{
		Name:     name,
		Email:    email,
		Password: password,
	}

	result, err := userService.Signup(context.Background(), req)
	if err != nil {
		fmt.Printf("Failed to create user: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("=== User Created Successfully! ===")
	fmt.Println()
	fmt.Printf("User ID:     %s\n", result.User.ID)
	fmt.Printf("Name:        %s\n", result.User.Name)
	fmt.Printf("Email:       %s\n", result.User.Email)
	fmt.Println()
	fmt.Println("=== Your API Key (save this securely!) ===")
	fmt.Println()
	fmt.Printf("  %s\n", result.APIKey)
	fmt.Println()
	fmt.Println("Use this API key in the X-API-Key header for all API requests.")
	fmt.Println()
}

func runListUsers() {
	// Connect to database
	cfg := config.Load()
	repository, err := repo.New(cfg.ResolveDatabaseDSN())
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		fmt.Println("Make sure the database is running and configured correctly.")
		os.Exit(1)
	}
	defer repository.Close()

	userService := service.NewUserService(repository)

	users, err := userService.ListUsers(context.Background())
	if err != nil {
		fmt.Printf("Failed to list users: %v\n", err)
		os.Exit(1)
	}

	if len(users) == 0 {
		fmt.Println("No users found.")
		return
	}

	fmt.Println("=== Users ===")
	fmt.Println()
	fmt.Printf("%-36s | %-20s | %-30s | %-6s\n", "ID", "Name", "Email", "Active")
	fmt.Println(strings.Repeat("-", 104))

	for _, user := range users {
		activeStr := "No"
		if user.Active {
			activeStr = "Yes"
		}
		fmt.Printf("%-36s | %-20s | %-30s | %-6s\n",
			user.ID,
			truncate(user.Name, 20),
			truncate(user.Email, 30),
			activeStr,
		)
	}
	fmt.Println()
	fmt.Printf("Total: %d user(s)\n", len(users))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
