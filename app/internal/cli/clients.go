package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/google/uuid"
	"github.com/martijn/dbcalm/internal/core/domain"
	"github.com/spf13/cobra"
)

var clientsCmd = &cobra.Command{
	Use:   "clients",
	Short: "Manage OAuth clients",
	Long:  "Manage OAuth client credentials for API authentication",
}

var clientsAddCmd = &cobra.Command{
	Use:   "add <label>",
	Short: "Add a new client",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		label := args[0]

		services, err := initServices(cmd.Context())
		if err != nil {
			return err
		}
		defer services.Close()

		// Generate secret
		secret := uuid.New().String()

		// Hash secret
		hashedSecret, err := services.AuthService.HashPassword(secret)
		if err != nil {
			return fmt.Errorf("failed to hash secret: %w", err)
		}

		// Create client
		client := domain.NewClient(label, hashedSecret, []string{"all"})
		if err := services.ClientRepo.Create(cmd.Context(), client); err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		fmt.Println("Client created successfully")
		fmt.Printf("Client ID: %s\n", client.ID)
		fmt.Printf("Client Secret: %s\n", secret)
		fmt.Println("\nIMPORTANT: Save the client secret now. It will not be shown again!")

		return nil
	},
}

var clientsDeleteCmd = &cobra.Command{
	Use:   "delete <client-id>",
	Short: "Delete a client",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		clientID := args[0]

		services, err := initServices(cmd.Context())
		if err != nil {
			return err
		}
		defer services.Close()

		// Confirm deletion
		fmt.Printf("Are you sure you want to delete client '%s'? (yes/no): ", clientID)
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			fmt.Println("Cancelled")
			return nil
		}

		if err := services.ClientRepo.Delete(cmd.Context(), clientID); err != nil {
			return fmt.Errorf("failed to delete client: %w", err)
		}

		fmt.Printf("Client '%s' deleted successfully\n", clientID)
		return nil
	},
}

var clientsUpdateCmd = &cobra.Command{
	Use:   "update <client-id> <new-label>",
	Short: "Update client label",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		clientID := args[0]
		newLabel := args[1]

		services, err := initServices(cmd.Context())
		if err != nil {
			return err
		}
		defer services.Close()

		// Get client
		client, err := services.ClientRepo.FindByID(cmd.Context(), clientID)
		if err != nil {
			return fmt.Errorf("client not found: %s", clientID)
		}

		// Update label
		client.Label = newLabel
		if err := services.ClientRepo.Update(cmd.Context(), client); err != nil {
			return fmt.Errorf("failed to update client: %w", err)
		}

		fmt.Printf("Client '%s' updated successfully\n", clientID)
		return nil
	},
}

var clientsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all clients",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := initServices(cmd.Context())
		if err != nil {
			return err
		}
		defer services.Close()

		clients, err := services.ClientRepo.List(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to list clients: %w", err)
		}

		if len(clients) == 0 {
			fmt.Println("No clients found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "CLIENT ID\tLABEL\tCREATED AT")
		for _, client := range clients {
			fmt.Fprintf(w, "%s\t%s\t%s\n",
				client.ID,
				client.Label,
				client.CreatedAt.Format("2006-01-02 15:04:05"),
			)
		}
		w.Flush()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(clientsCmd)
	clientsCmd.AddCommand(clientsAddCmd)
	clientsCmd.AddCommand(clientsDeleteCmd)
	clientsCmd.AddCommand(clientsUpdateCmd)
	clientsCmd.AddCommand(clientsListCmd)
}
