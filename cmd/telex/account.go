package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/elpdev/telex-cli/internal/api"
	"github.com/elpdev/telex-cli/internal/config"
	"github.com/spf13/cobra"
)

func newAccountCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "account", Short: "Account commands"}
	auth := &cobra.Command{Use: "auth", Short: "Authentication commands"}
	login := newAuthLoginCommand(rt)
	auth.AddCommand(login)
	cmd.AddCommand(auth)
	return cmd
}

func newAuthLoginCommand(rt *runtime) *cobra.Command {
	var baseURL, clientID, secretKey string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Set up API credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(cmd.InOrStdin())
			if baseURL == "" {
				baseURL = prompt(reader, cmd.OutOrStdout(), "Base URL", "http://localhost:3000")
			}
			if clientID == "" {
				clientID = prompt(reader, cmd.OutOrStdout(), "Client ID", "")
			}
			if secretKey == "" {
				secretKey = prompt(reader, cmd.OutOrStdout(), "Secret Key", "")
			}
			cfg := &config.Config{BaseURL: baseURL, ClientID: clientID, SecretKey: secretKey}
			if err := cfg.Validate(); err != nil {
				return err
			}
			configFile, tokenFile := rt.configFiles()
			client := api.NewClient(cfg, tokenFile)
			if err := client.Authenticate(rt.context()); err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}
			if err := cfg.SaveTo(configFile); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Config saved to %s\n", configFile)
			fmt.Fprintf(cmd.OutOrStdout(), "Token cache saved to %s\n", tokenFile)
			rt.client = nil
			return nil
		},
	}
	cmd.Flags().StringVar(&baseURL, "base-url", "", "Telex server URL")
	cmd.Flags().StringVar(&clientID, "client-id", "", "API key client_id")
	cmd.Flags().StringVar(&secretKey, "secret-key", "", "API key secret")
	return cmd
}

func prompt(reader *bufio.Reader, out io.Writer, label, defaultValue string) string {
	if defaultValue == "" {
		fmt.Fprintf(out, "%s: ", label)
	} else {
		fmt.Fprintf(out, "%s [%s]: ", label, defaultValue)
	}
	value, _ := reader.ReadString('\n')
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultValue
	}
	return value
}
