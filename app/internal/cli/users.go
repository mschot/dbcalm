package cli

import (
	"fmt"
	"os"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/martijn/dbcalm/internal/core/domain"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Manage users",
	Long:  "Manage user accounts for authentication",
}

var usersAddCmd = &cobra.Command{
	Use:   "add <username>",
	Short: "Add a new user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]

		services, err := initServices(cmd.Context())
		if err != nil {
			return err
		}
		defer services.Close()

		// Check if user already exists
		_, err = services.UserRepo.FindByUsername(cmd.Context(), username)
		if err == nil {
			return fmt.Errorf("user already exists: %s", username)
		}

		// Prompt for password
		fmt.Print("Enter password: ")
		password, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}

		fmt.Print("Confirm password: ")
		confirmPassword, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}

		if string(password) != string(confirmPassword) {
			return fmt.Errorf("passwords do not match")
		}

		if len(password) < 8 {
			return fmt.Errorf("password must be at least 8 characters")
		}

		// Hash password
		hashedPassword, err := services.AuthService.HashPassword(string(password))
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}

		// Create user
		user := domain.NewUser(username, hashedPassword)
		if err := services.UserRepo.Create(cmd.Context(), user); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		fmt.Printf("User '%s' created successfully\n", username)
		return nil
	},
}

var usersDeleteCmd = &cobra.Command{
	Use:   "delete <username>",
	Short: "Delete a user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]

		services, err := initServices(cmd.Context())
		if err != nil {
			return err
		}
		defer services.Close()

		// Confirm deletion
		fmt.Printf("Are you sure you want to delete user '%s'? (yes/no): ", username)
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			fmt.Println("Cancelled")
			return nil
		}

		if err := services.UserRepo.Delete(cmd.Context(), username); err != nil {
			return fmt.Errorf("failed to delete user: %w", err)
		}

		fmt.Printf("User '%s' deleted successfully\n", username)
		return nil
	},
}

var usersUpdatePasswordCmd = &cobra.Command{
	Use:   "update-password <username>",
	Short: "Update user password",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]

		services, err := initServices(cmd.Context())
		if err != nil {
			return err
		}
		defer services.Close()

		// Check if user exists
		user, err := services.UserRepo.FindByUsername(cmd.Context(), username)
		if err != nil {
			return fmt.Errorf("user not found: %s", username)
		}

		// Prompt for new password
		fmt.Print("Enter new password: ")
		password, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}

		fmt.Print("Confirm new password: ")
		confirmPassword, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}

		if string(password) != string(confirmPassword) {
			return fmt.Errorf("passwords do not match")
		}

		if len(password) < 8 {
			return fmt.Errorf("password must be at least 8 characters")
		}

		// Hash password
		hashedPassword, err := services.AuthService.HashPassword(string(password))
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}

		// Update user
		user.Password = hashedPassword
		user.UpdatedAt = time.Now()
		if err := services.UserRepo.Update(cmd.Context(), user); err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

		fmt.Printf("Password updated for user '%s'\n", username)
		return nil
	},
}

var usersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := initServices(cmd.Context())
		if err != nil {
			return err
		}
		defer services.Close()

		users, err := services.UserRepo.List(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}

		if len(users) == 0 {
			fmt.Println("No users found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "USERNAME\tCREATED AT\tUPDATED AT")
		for _, user := range users {
			fmt.Fprintf(w, "%s\t%s\t%s\n",
				user.Username,
				user.CreatedAt.Format("2006-01-02 15:04:05"),
				user.UpdatedAt.Format("2006-01-02 15:04:05"),
			)
		}
		w.Flush()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(usersCmd)
	usersCmd.AddCommand(usersAddCmd)
	usersCmd.AddCommand(usersDeleteCmd)
	usersCmd.AddCommand(usersUpdatePasswordCmd)
	usersCmd.AddCommand(usersListCmd)
}
