package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/elpdev/telex-cli/internal/mailstore"
	"github.com/spf13/cobra"
)

func newDraftsCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "drafts", Short: "Manage drafts"}
	cmd.AddCommand(newDraftCreateCommand(rt))
	cmd.AddCommand(newDraftListCommand(rt))
	cmd.AddCommand(newDraftShowCommand(rt))
	cmd.AddCommand(newDraftEditCommand(rt))
	cmd.AddCommand(newDraftAttachCommand(rt))
	cmd.AddCommand(newDraftDetachCommand(rt))
	cmd.AddCommand(newDraftDeleteCommand(rt))
	cmd.AddCommand(newDraftSendCommand(rt))
	return cmd
}

func newDraftCreateCommand(rt *runtime) *cobra.Command {
	var mailboxAddress string
	var subject string
	var to []string
	var cc []string
	var bcc []string
	var body string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a draft",
		RunE: func(cmd *cobra.Command, args []string) error {
			store := mailstore.New(rt.dataPath)
			mailbox, _, err := store.FindMailboxByAddress(mailboxAddress)
			if err != nil {
				return err
			}
			draft, err := store.CreateDraft(mailstore.DraftInput{
				Mailbox: *mailbox,
				Subject: subject,
				To:      splitAddresses(to),
				CC:      splitAddresses(cc),
				BCC:     splitAddresses(bcc),
				Body:    body,
				Now:     time.Now(),
			})
			if err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, [][]string{
				{"id", draft.Meta.ID},
				{"from", draft.Meta.FromAddress},
				{"subject", draft.Meta.Subject},
				{"path", draft.Path},
			})
			return nil
		},
	}
	cmd.Flags().StringVar(&mailboxAddress, "mailbox", "", "synced mailbox address, e.g. hello@example.com")
	cmd.Flags().StringVar(&subject, "subject", "", "draft subject")
	cmd.Flags().StringSliceVar(&to, "to", nil, "recipient address, repeatable or comma-separated")
	cmd.Flags().StringSliceVar(&cc, "cc", nil, "cc address, repeatable or comma-separated")
	cmd.Flags().StringSliceVar(&bcc, "bcc", nil, "bcc address, repeatable or comma-separated")
	cmd.Flags().StringVar(&body, "body", "", "initial Markdown body")
	_ = cmd.MarkFlagRequired("mailbox")
	return cmd
}

func newDraftListCommand(rt *runtime) *cobra.Command {
	var mailboxAddress string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List drafts for a synced mailbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			store := mailstore.New(rt.dataPath)
			_, mailboxPath, err := store.FindMailboxByAddress(mailboxAddress)
			if err != nil {
				return err
			}
			drafts, err := mailstore.ListDrafts(mailboxPath)
			if err != nil {
				return err
			}
			rows := make([][]string, 0, len(drafts))
			for _, draft := range drafts {
				rows = append(rows, []string{
					draft.Meta.ID,
					draft.Meta.Subject,
					strings.Join(draft.Meta.To, ", "),
					draft.Meta.UpdatedAt.Format("2006-01-02 15:04"),
					draft.Path,
				})
			}
			writeRows(cmd.OutOrStdout(), []string{"id", "subject", "to", "updated_at", "path"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&mailboxAddress, "mailbox", "", "synced mailbox address, e.g. hello@example.com")
	_ = cmd.MarkFlagRequired("mailbox")
	return cmd
}

func newDraftShowCommand(rt *runtime) *cobra.Command {
	var mailboxAddress string
	var latest bool
	cmd := &cobra.Command{
		Use:   "show [draft-id]",
		Short: "Show a draft",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := mailstore.New(rt.dataPath)
			_, mailboxPath, err := store.FindMailboxByAddress(mailboxAddress)
			if err != nil {
				return err
			}
			draftID, err := resolveDraftID(mailboxAddress, mailboxPath, args, latest)
			if err != nil {
				return err
			}
			draft, err := mailstore.ReadDraft(filepath.Join(mailboxPath, "drafts", draftID))
			if err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, draftFields(*draft))
			fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", draft.Body)
			return nil
		},
	}
	cmd.Flags().StringVar(&mailboxAddress, "mailbox", "", "synced mailbox address, e.g. hello@example.com")
	cmd.Flags().BoolVar(&latest, "latest", false, "show the newest draft")
	_ = cmd.MarkFlagRequired("mailbox")
	return cmd
}

func newDraftEditCommand(rt *runtime) *cobra.Command {
	var mailboxAddress string
	var latest bool
	var subject string
	var to []string
	var cc []string
	var bcc []string
	var body string
	cmd := &cobra.Command{
		Use:   "edit [draft-id]",
		Short: "Edit draft fields",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := mailstore.New(rt.dataPath)
			mailbox, mailboxPath, err := store.FindMailboxByAddress(mailboxAddress)
			if err != nil {
				return err
			}
			draftID, err := resolveDraftID(mailboxAddress, mailboxPath, args, latest)
			if err != nil {
				return err
			}
			draftPath := filepath.Join(mailboxPath, "drafts", draftID)
			draft, err := mailstore.ReadDraft(draftPath)
			if err != nil {
				return err
			}
			input := mailstore.DraftInput{Mailbox: *mailbox, Subject: draft.Meta.Subject, To: draft.Meta.To, CC: draft.Meta.CC, BCC: draft.Meta.BCC, Body: draft.Body, SourceMessageID: draft.Meta.SourceMessageID, ConversationID: draft.Meta.ConversationID, Now: time.Now()}
			if cmd.Flags().Changed("subject") {
				input.Subject = subject
			}
			if cmd.Flags().Changed("to") {
				input.To = splitAddresses(to)
			}
			if cmd.Flags().Changed("cc") {
				input.CC = splitAddresses(cc)
			}
			if cmd.Flags().Changed("bcc") {
				input.BCC = splitAddresses(bcc)
			}
			if cmd.Flags().Changed("body") {
				input.Body = body
			}
			updated, err := store.UpdateDraft(draftPath, input)
			if err != nil {
				return err
			}
			if updated.Meta.RemoteID > 0 {
				service, err := mailService(rt)
				if err != nil {
					return err
				}
				if _, err := service.UpdateOutboundMessage(rt.context(), updated.Meta.RemoteID, outboundInputFromDraft(*updated)); err != nil {
					return err
				}
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, draftFields(*updated))
			return nil
		},
	}
	cmd.Flags().StringVar(&mailboxAddress, "mailbox", "", "synced mailbox address, e.g. hello@example.com")
	cmd.Flags().BoolVar(&latest, "latest", false, "edit the newest draft")
	cmd.Flags().StringVar(&subject, "subject", "", "draft subject")
	cmd.Flags().StringSliceVar(&to, "to", nil, "recipient address, repeatable or comma-separated")
	cmd.Flags().StringSliceVar(&cc, "cc", nil, "cc address, repeatable or comma-separated")
	cmd.Flags().StringSliceVar(&bcc, "bcc", nil, "bcc address, repeatable or comma-separated")
	cmd.Flags().StringVar(&body, "body", "", "Markdown body")
	_ = cmd.MarkFlagRequired("mailbox")
	return cmd
}
