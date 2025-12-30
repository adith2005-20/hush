package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/adith2005-20/hush/pkg/client"
	"github.com/adith2005-20/hush/pkg/config"
	"github.com/adith2005-20/hush/pkg/crypto"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "hush",
	Short: "Hush - Git for secrets",
	Long:  `A lightweight secrets manager for developers and homelabs`,
}

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new Hush project",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		server, _ := cmd.Flags().GetString("server")
		env, _ := cmd.Flags().GetString("env")

		cfg := &config.Config{
			Project:     args[0],
			Server:      server,
			Environment: env,
			Output: config.OutputConfig{
				Format: "dotenv",
				Path:   ".env",
			},
		}

		if err := cfg.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Generate master key if it doesn't exist
		if _, err := config.LoadMasterKey(); err != nil {
			salt, _ := crypto.GenerateSalt()
			if err := config.SaveMasterKey(salt); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating master key: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✓ Generated master encryption key")
		}

		fmt.Printf("✓ Initialized Hush project: %s\n", args[0])
		fmt.Printf("✓ Created .hush configuration file\n")
		fmt.Println("\nNext steps:")
		fmt.Println("  hush set KEY=value    # Add a secret")
		fmt.Println("  hush push             # Push secrets to server")
	},
}

var setCmd = &cobra.Command{
	Use:   "set KEY=VALUE",
	Short: "Set a secret (encrypts locally)",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Println("Run 'hush init' first")
			os.Exit(1)
		}

		masterKey, err := config.LoadMasterKey()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading master key: %v\n", err)
			os.Exit(1)
		}

		token := os.Getenv("HUSH_TOKEN")
		if token == "" {
			fmt.Fprintf(os.Stderr, "Error: HUSH_TOKEN environment variable not set\n")
			os.Exit(1)
		}

		cli := client.New(cfg.Server, token)

		for _, arg := range args {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) != 2 {
				fmt.Fprintf(os.Stderr, "Invalid format: %s (use KEY=VALUE)\n", arg)
				continue
			}

			key, value := parts[0], parts[1]
			encrypted, err := crypto.Encrypt(value, masterKey)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Encryption error: %v\n", err)
				continue
			}

			if err := cli.SetSecret(cfg.Project, cfg.Environment, key, encrypted); err != nil {
				fmt.Fprintf(os.Stderr, "Error setting %s: %v\n", key, err)
				continue
			}

			fmt.Printf("✓ Set %s\n", key)
		}
	},
}

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull secrets from server and write to output file",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		masterKey, err := config.LoadMasterKey()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading master key: %v\n", err)
			os.Exit(1)
		}

		token := os.Getenv("HUSH_TOKEN")
		if token == "" {
			fmt.Fprintf(os.Stderr, "Error: HUSH_TOKEN environment variable not set\n")
			os.Exit(1)
		}

		cli := client.New(cfg.Server, token)
		secrets, err := cli.GetSecrets(cfg.Project, cfg.Environment)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching secrets: %v\n", err)
			os.Exit(1)
		}

		// Decrypt and write to file
		var output strings.Builder
		for _, secret := range secrets {
			decrypted, err := crypto.Decrypt(secret.Value, masterKey)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error decrypting %s: %v\n", secret.Key, err)
				continue
			}
			output.WriteString(fmt.Sprintf("%s%s=%s\n", cfg.Prefix, secret.Key, decrypted))
		}

		if err := os.WriteFile(cfg.Output.Path, []byte(output.String()), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to %s: %v\n", cfg.Output.Path, err)
			os.Exit(1)
		}

		fmt.Printf("✓ Pulled %d secrets to %s\n", len(secrets), cfg.Output.Path)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all secrets (keys only)",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		token := os.Getenv("HUSH_TOKEN")
		if token == "" {
			fmt.Fprintf(os.Stderr, "Error: HUSH_TOKEN environment variable not set\n")
			os.Exit(1)
		}

		cli := client.New(cfg.Server, token)
		secrets, err := cli.GetSecrets(cfg.Project, cfg.Environment)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Secrets for %s/%s:\n", cfg.Project, cfg.Environment)
		for _, secret := range secrets {
			fmt.Printf("  • %s\n", secret.Key)
		}
	},
}

func init() {
	initCmd.Flags().String("server", "http://localhost:8080", "Hush server URL")
	initCmd.Flags().String("env", "production", "Environment name")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(listCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
