package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/elpdev/telex-cli/internal/mailstore"
	"github.com/spf13/cobra"
)

func newInboxCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "inbox", Short: "Read cached inbox messages"}
	cmd.AddCommand(newInboxListCommand(rt))
	cmd.AddCommand(newInboxShowCommand(rt))
	cmd.AddCommand(newInboxForwardCommand(rt))
	return cmd
}

func newInboxListCommand(rt *runtime) *cobra.Command {
	var mailboxAddress string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List cached inbox messages for a synced mailbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			store := mailstore.New(rt.dataPath)
			_, mailboxPath, err := store.FindMailboxByAddress(mailboxAddress)
			if err != nil {
				return err
			}
			messages, err := mailstore.ListInbox(mailboxPath)
			if err != nil {
				return err
			}
			rows := make([][]string, 0, len(messages))
			for _, message := range messages {
				rows = append(rows, cachedMessageRow(message))
			}
			writeRows(cmd.OutOrStdout(), []string{"id", "from", "subject", "read", "starred", "received_at", "path"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&mailboxAddress, "mailbox", "", "synced mailbox address, e.g. hello@example.com")
	_ = cmd.MarkFlagRequired("mailbox")
	return cmd
}

func newInboxShowCommand(rt *runtime) *cobra.Command {
	var mailboxAddress string
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show a cached inbox message",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			store := mailstore.New(rt.dataPath)
			_, mailboxPath, err := store.FindMailboxByAddress(mailboxAddress)
			if err != nil {
				return err
			}
			message, err := mailstore.FindInboxMessage(mailboxPath, id)
			if err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, cachedMessageFields(*message))
			content := strings.TrimSpace(message.BodyText)
			if content == "" {
				content = strings.TrimSpace(message.BodyHTML)
			}
			if content == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "\n(body not cached)")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", content)
			return nil
		},
	}
	cmd.Flags().StringVar(&mailboxAddress, "mailbox", "", "synced mailbox address, e.g. hello@example.com")
	_ = cmd.MarkFlagRequired("mailbox")
	return cmd
}

func newInboxForwardCommand(rt *runtime) *cobra.Command {
	var mailboxAddress string
	var to []string
	cmd := &cobra.Command{
		Use:   "forward <id>",
		Short: "Create a remote forward draft from a cached inbox message",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			store := mailstore.New(rt.dataPath)
			_, mailboxPath, err := store.FindMailboxByAddress(mailboxAddress)
			if err != nil {
				return err
			}
			message, err := mailstore.FindInboxMessage(mailboxPath, id)
			if err != nil {
				return err
			}
			recipients := splitAddresses(to)
			if len(recipients) == 0 {
				return fmt.Errorf("at least one --to recipient is required")
			}
			service, err := mailService(rt)
			if err != nil {
				return err
			}
			outbound, err := service.Forward(rt.context(), message.Meta.RemoteID, recipients)
			if err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, [][]string{
				{"remote_id", strconv.FormatInt(outbound.ID, 10)},
				{"status", outbound.Status},
				{"subject", outbound.Subject},
				{"to", strings.Join(outbound.ToAddresses, ", ")},
			})
			fmt.Fprintln(cmd.OutOrStdout(), "Forward draft created remotely. It has not been sent; review and send it from drafts/web.")
			return nil
		},
	}
	cmd.Flags().StringVar(&mailboxAddress, "mailbox", "", "synced mailbox address, e.g. hello@example.com")
	cmd.Flags().StringSliceVar(&to, "to", nil, "forward recipient address, repeatable or comma-separated")
	_ = cmd.MarkFlagRequired("mailbox")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}
