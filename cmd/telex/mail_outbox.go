package main

import (
	"strconv"

	"github.com/elpdev/telex-cli/internal/mailstore"
	"github.com/elpdev/telex-cli/internal/mailsync"
	"github.com/spf13/cobra"
)

func newOutboxCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "outbox", Short: "Manage queued local outbound messages"}
	cmd.AddCommand(newOutboxListCommand(rt))
	cmd.AddCommand(newOutboxSyncCommand(rt))
	return cmd
}

func newOutboxListCommand(rt *runtime) *cobra.Command {
	var mailboxAddress string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List local outbox items for a synced mailbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			store := mailstore.New(rt.dataPath)
			_, mailboxPath, err := store.FindMailboxByAddress(mailboxAddress)
			if err != nil {
				return err
			}
			items, err := mailstore.ListOutbox(mailboxPath)
			if err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"remote_id", "status", "subject", "to", "updated_at", "path"}, draftRows(items))
			return nil
		},
	}
	cmd.Flags().StringVar(&mailboxAddress, "mailbox", "", "synced mailbox address, e.g. hello@example.com")
	_ = cmd.MarkFlagRequired("mailbox")
	return cmd
}

func newOutboxSyncCommand(rt *runtime) *cobra.Command {
	var mailboxAddress string
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync local outbox items with remote delivery status",
		RunE: func(cmd *cobra.Command, args []string) error {
			store := mailstore.New(rt.dataPath)
			mailbox, _, err := store.FindMailboxByAddress(mailboxAddress)
			if err != nil {
				return err
			}
			service, err := mailService(rt)
			if err != nil {
				return err
			}
			updates, err := mailsync.SyncOutboxForMailbox(rt.context(), service, store, *mailbox)
			if err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"remote_id", "status", "subject", "path"}, outboxUpdateRows(updates, false))
			return nil
		},
	}
	cmd.Flags().StringVar(&mailboxAddress, "mailbox", "", "synced mailbox address, e.g. hello@example.com")
	_ = cmd.MarkFlagRequired("mailbox")
	return cmd
}

func outboxUpdateRows(updates []mailsync.OutboxUpdate, includeMailbox bool) [][]string {
	rows := make([][]string, 0, len(updates))
	for _, update := range updates {
		row := []string{strconv.FormatInt(update.RemoteID, 10), update.Status, update.Subject, update.Path}
		if includeMailbox {
			row = append([]string{update.Mailbox}, row...)
		}
		rows = append(rows, row)
	}
	return rows
}
