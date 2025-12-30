package main

import (
    "fmt"
    "os"
    "strings"
    
    "github.com/spf13/cobra"
    "github.com/adith2005-20/hush/pkg/config"
    "github.com/adith2005-20/hush/pkg/crypto"
    "github.com/adith2005-20/hush/pkg/client"
)

var rootCmd = &cobra.Command{
    Use:   "hush",
    Short: "Hush - Secrets manager for developers",
}

var loginCmd = &cobra.Command{
    Use:   "login [server-url] [token]",
    Short: "Login to a Hush server",
    Long: `Authenticate with a Hush server and save credentials.

Examples:
  hush login http://localhost:555555 hush_abc123
  hush login https://secrets.mycompany.com hush_def456`,
    Args: cobra.ExactArgs(2),
    Run: func(cmd *cobra.Command, args []string) {
        server := args[0]
        token := args[1]
        
        fmt.Printf("🔐 Connecting to %s...\n", server)
        
        // Test connection
        cli := client.New(server, token)
        if err := cli.Ping(); err != nil {
            fmt.Printf("❌ Failed to connect: %v\n", err)
            os.Exit(1)
        }
        
        // Save credentials
        if err := config.SaveCredentials(server, token); err != nil {
            fmt.Printf("❌ Failed to save credentials: %v\n", err)
            os.Exit(1)
        }
        
        // Generate master key if doesn't exist
        if _, err := config.LoadMasterKey(); err != nil {
            salt, _ := crypto.GenerateSalt()
            if err := config.SaveMasterKey(salt); err != nil {
                fmt.Printf("❌ Failed to create master key: %v\n", err)
                os.Exit(1)
            }
            fmt.Println("✓ Generated master encryption key")
        }
        
        fmt.Println("✓ Authenticated successfully!")
        fmt.Println()
        fmt.Println("Next steps:")
        fmt.Println("  hush init myproject    # Initialize a project")
    },
}

var initCmd = &cobra.Command{
    Use:   "init [project-name]",
    Short: "Initialize a new Hush project",
    Args:  cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        // Check if logged in
        creds, err := config.LoadCredentials()
        if err != nil {
            fmt.Println("❌ Not logged in!")
            fmt.Println("\nRun this first:")
            fmt.Println("  hush login http://server:555555 your-token")
            os.Exit(1)
        }
        
        env, _ := cmd.Flags().GetString("env")

        cfg := &config.Config{
            Project:     args[0],
            Server:      creds.Server,
            Environment: env,
            Output: config.OutputConfig{
                Format: "dotenv",
                Path:   ".env",
            },
        }

        if err := cfg.Save(); err != nil {
            fmt.Printf("❌ Error: %v\n", err)
            os.Exit(1)
        }

        fmt.Printf("✓ Initialized project: %s\n", args[0])
        fmt.Printf("✓ Created hush.yaml\n")
        fmt.Printf("✓ Environment: %s\n", env)
        fmt.Println()
        fmt.Println("Next steps:")
        fmt.Println("  hush set KEY=value    # Add secrets")
        fmt.Println("  hush list             # View secrets")
        fmt.Println("  hush pull             # Download to .env")
    },
}

var setCmd = &cobra.Command{
    Use:   "set KEY=VALUE [KEY2=VALUE2 ...]",
    Short: "Set one or more secrets",
    Args:  cobra.MinimumNArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        cfg, err := config.LoadProjectConfig()
        if err != nil {
            fmt.Printf("❌ %v\n", err)
            os.Exit(1)
        }

        creds, err := config.LoadCredentials()
        if err != nil {
            fmt.Printf("❌ %v\n", err)
            os.Exit(1)
        }

        masterKey, err := config.LoadMasterKey()
        if err != nil {
            fmt.Printf("❌ Error loading encryption key: %v\n", err)
            os.Exit(1)
        }

        cli := client.New(creds.Server, creds.Token)

        for _, arg := range args {
            parts := strings.SplitN(arg, "=", 2)
            if len(parts) != 2 {
                fmt.Printf("⚠️  Invalid format: %s (use KEY=VALUE)\n", arg)
                continue
            }

            key, value := parts[0], parts[1]
            encrypted, err := crypto.Encrypt(value, masterKey)
            if err != nil {
                fmt.Printf("❌ Encryption error for %s: %v\n", key, err)
                continue
            }

            if err := cli.SetSecret(cfg.Project, cfg.Environment, key, encrypted); err != nil {
                fmt.Printf("❌ Error setting %s: %v\n", key, err)
                continue
            }

            fmt.Printf("✓ Set %s\n", key)
        }
    },
}

var pullCmd = &cobra.Command{
    Use:   "pull",
    Short: "Pull secrets and write to output file",
    Run: func(cmd *cobra.Command, args []string) {
        cfg, err := config.LoadProjectConfig()
        if err != nil {
            fmt.Printf("❌ %v\n", err)
            os.Exit(1)
        }

        creds, err := config.LoadCredentials()
        if err != nil {
            fmt.Printf("❌ %v\n", err)
            os.Exit(1)
        }

        masterKey, err := config.LoadMasterKey()
        if err != nil {
            fmt.Printf("❌ Error loading encryption key: %v\n", err)
            os.Exit(1)
        }

        cli := client.New(creds.Server, creds.Token)
        secrets, err := cli.GetSecrets(cfg.Project, cfg.Environment)
        if err != nil {
            fmt.Printf("❌ Error fetching secrets: %v\n", err)
            os.Exit(1)
        }

        if len(secrets) == 0 {
            fmt.Println("⚠️  No secrets found")
            fmt.Println("\nAdd secrets with:")
            fmt.Println("  hush set KEY=value")
            return
        }

        var output strings.Builder
        for _, secret := range secrets {
            decrypted, err := crypto.Decrypt(secret.Value, masterKey)
            if err != nil {
                fmt.Printf("❌ Error decrypting %s: %v\n", secret.Key, err)
                continue
            }
            output.WriteString(fmt.Sprintf("%s%s=%s\n", cfg.Prefix, secret.Key, decrypted))
        }

        if err := os.WriteFile(cfg.Output.Path, []byte(output.String()), 0600); err != nil {
            fmt.Printf("❌ Error writing to %s: %v\n", cfg.Output.Path, err)
            os.Exit(1)
        }

        fmt.Printf("✓ Pulled %d secrets to %s\n", len(secrets), cfg.Output.Path)
    },
}

var listCmd = &cobra.Command{
    Use:   "list",
    Short: "List all secrets (keys only)",
    Run: func(cmd *cobra.Command, args []string) {
        cfg, err := config.LoadProjectConfig()
        if err != nil {
            fmt.Printf("❌ %v\n", err)
            os.Exit(1)
        }

        creds, err := config.LoadCredentials()
        if err != nil {
            fmt.Printf("❌ %v\n", err)
            os.Exit(1)
        }

        cli := client.New(creds.Server, creds.Token)
        secrets, err := cli.GetSecrets(cfg.Project, cfg.Environment)
        if err != nil {
            fmt.Printf("❌ Error: %v\n", err)
            os.Exit(1)
        }

        if len(secrets) == 0 {
            fmt.Println("No secrets found")
            return
        }

        fmt.Printf("Secrets for %s/%s:\n", cfg.Project, cfg.Environment)
        for _, secret := range secrets {
            fmt.Printf("  • %s\n", secret.Key)
        }
    },
}

func init() {
    initCmd.Flags().String("env", "production", "Environment name")

    rootCmd.AddCommand(loginCmd)
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