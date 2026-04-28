package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/elpdev/telex-cli/internal/mailstore"
	"github.com/elpdev/telex-cli/internal/mailsync"
	"github.com/spf13/cobra"
)

func newMailboxesCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mailboxes",
		Short: "Show mailbox bootstrap data",
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := mailService(rt)
			if err != nil {
				return err
			}
			bootstrap, err := service.Mailboxes(rt.context())
			if err != nil {
				return err
			}
			rows := [][]string{
				{"inbox", strconv.Itoa(bootstrap.Counts.Inbox)},
				{"junk", strconv.Itoa(bootstrap.Counts.Junk)},
				{"archived", strconv.Itoa(bootstrap.Counts.Archived)},
				{"trash", strconv.Itoa(bootstrap.Counts.Trash)},
				{"sent", strconv.Itoa(bootstrap.Counts.Sent)},
				{"drafts", strconv.Itoa(bootstrap.Counts.Drafts)},
				{"inboxes", strings.Join(inboxAddresses(bootstrap.Inboxes), ", ")},
				{"domains", strings.Join(domainNames(bootstrap.Domains), ", ")},
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, rows)
			return nil
		},
	}
	cmd.AddCommand(newMailboxesSyncCommand(rt))
	return cmd
}

func newMailboxesSyncCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync active mailbox folders to the local filesystem",
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := mailService(rt)
			if err != nil {
				return err
			}
			store := mailstore.New(rt.dataPath)
			result, err := mailsync.SyncMailboxes(rt.context(), store, service)
			if err != nil {
				return err
			}
			rows := make([][]string, 0, len(result.Created))
			for _, meta := range result.Created {
				path, err := store.MailboxPath(meta.DomainName, meta.LocalPart)
				if err != nil {
					return err
				}
				rows = append(rows, []string{meta.DomainName, meta.LocalPart, meta.Address, path})
			}
			writeRows(cmd.OutOrStdout(), []string{"domain", "mailbox", "address", "path"}, rows)
			if len(result.Skipped) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Skipped %d inactive mailbox(es).\n", len(result.Skipped))
			}
			return nil
		},
	}
}
