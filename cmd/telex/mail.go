package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/elpdev/telex-cli/internal/mail"
	"github.com/elpdev/telex-cli/internal/mailsend"
	"github.com/elpdev/telex-cli/internal/mailstore"
	"github.com/elpdev/telex-cli/internal/mailsync"
	"github.com/spf13/cobra"
)

func newMailCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "mail", Short: "Email commands"}
	cmd.AddCommand(newMailSyncCommand(rt))
	cmd.AddCommand(newLabelsCommand(rt))
	cmd.AddCommand(newMailboxesCommand(rt))
	cmd.AddCommand(newInboxCommand(rt))
	cmd.AddCommand(newDraftsCommand(rt))
	cmd.AddCommand(newOutboxCommand(rt))
	cmd.AddCommand(newMailSearchCommand(rt))
	cmd.AddCommand(newConversationsCommand(rt))
	cmd.AddCommand(newMessagesCommand(rt))
	return cmd
}

func newLabelsCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "labels", Short: "Manage remote labels"}
	cmd.AddCommand(newLabelsListCommand(rt))
	return cmd
}

func newLabelsListCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List remote labels",
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := mailService(rt)
			if err != nil {
				return err
			}
			labels, err := service.Labels(rt.context())
			if err != nil {
				return err
			}
			rows := make([][]string, 0, len(labels))
			for _, label := range labels {
				rows = append(rows, labelRow(label))
			}
			writeRows(cmd.OutOrStdout(), []string{"id", "name", "color"}, rows)
			return nil
		},
	}
}

func newMailSearchCommand(rt *runtime) *cobra.Command {
	var params mail.MessageListParams
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search remote messages",
		Long:  "Search remote messages. The query matches sender, recipients, subject, body text, and attachment filenames.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params.Query = args[0]
			return runMessageList(cmd, rt, params)
		},
	}
	addMessageListFlags(cmd, &params)
	return cmd
}

func newMailSyncCommand(rt *runtime) *cobra.Command {
	var mailboxAddress string
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync local mail data",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMailSync(cmd, rt, mailboxAddress)
		},
	}
	cmd.Flags().StringVar(&mailboxAddress, "mailbox", "", "limit sync to one synced mailbox address")
	return cmd
}

func runMailSync(cmd *cobra.Command, rt *runtime, mailboxAddress string) error {
	service, err := mailService(rt)
	if err != nil {
		return err
	}
	store := mailstore.New(rt.dataPath)
	result, err := mailsync.Run(rt.context(), store, service, mailboxAddress)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Synced %d active mailbox(es).\n", result.ActiveMailboxes)
	if result.SkippedMailboxes > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Skipped %d inactive mailbox(es).\n", result.SkippedMailboxes)
	}
	rows := outboxUpdateRows(result.OutboxUpdates, true)
	if len(rows) > 0 {
		writeRows(cmd.OutOrStdout(), []string{"mailbox", "remote_id", "status", "subject", "path"}, rows)
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "Outbox already synced.")
	}
	if result.DraftItems > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Synced %d remote draft(s).\n", result.DraftItems)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Synced %d inbox message(s).\n", result.InboxMessages)
	if result.BodyErrors > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "Skipped %d inbox message body fetch(es) due to remote API errors; metadata was still cached.\n", result.BodyErrors)
	}
	if result.InboxErrors > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "Skipped partial inbox sync for %d mailbox(es) due to remote API errors.\n", result.InboxErrors)
	}
	return nil
}

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

func newDraftsCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "drafts", Short: "Manage local drafts"}
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
		Short: "Create a local draft",
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
		Short: "List local drafts for a synced mailbox",
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
		Short: "Show a local draft",
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
	cmd.Flags().BoolVar(&latest, "latest", false, "show the newest local draft")
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
		Short: "Edit local draft fields",
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
	cmd.Flags().BoolVar(&latest, "latest", false, "edit the newest local draft")
	cmd.Flags().StringVar(&subject, "subject", "", "draft subject")
	cmd.Flags().StringSliceVar(&to, "to", nil, "recipient address, repeatable or comma-separated")
	cmd.Flags().StringSliceVar(&cc, "cc", nil, "cc address, repeatable or comma-separated")
	cmd.Flags().StringSliceVar(&bcc, "bcc", nil, "bcc address, repeatable or comma-separated")
	cmd.Flags().StringVar(&body, "body", "", "Markdown body")
	_ = cmd.MarkFlagRequired("mailbox")
	return cmd
}

func newDraftAttachCommand(rt *runtime) *cobra.Command {
	var mailboxAddress string
	var latest bool
	cmd := &cobra.Command{
		Use:   "attach [draft-id] <file>",
		Short: "Attach a local file to a local draft",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := mailstore.New(rt.dataPath)
			_, mailboxPath, err := store.FindMailboxByAddress(mailboxAddress)
			if err != nil {
				return err
			}
			draftArgs := args[:len(args)-1]
			filePath := args[len(args)-1]
			draftID, err := resolveDraftID(mailboxAddress, mailboxPath, draftArgs, latest)
			if err != nil {
				return err
			}
			draft, err := mailstore.AttachFileToDraft(filepath.Join(mailboxPath, "drafts", draftID), filePath, time.Now())
			if err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, draftFields(*draft))
			return nil
		},
	}
	cmd.Flags().StringVar(&mailboxAddress, "mailbox", "", "synced mailbox address, e.g. hello@example.com")
	cmd.Flags().BoolVar(&latest, "latest", false, "attach to the newest local draft")
	_ = cmd.MarkFlagRequired("mailbox")
	return cmd
}

func newDraftDetachCommand(rt *runtime) *cobra.Command {
	var mailboxAddress string
	var latest bool
	cmd := &cobra.Command{
		Use:   "detach [draft-id] <attachment>",
		Short: "Detach a local file from a local draft",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := mailstore.New(rt.dataPath)
			_, mailboxPath, err := store.FindMailboxByAddress(mailboxAddress)
			if err != nil {
				return err
			}
			draftArgs := args[:len(args)-1]
			attachmentName := args[len(args)-1]
			draftID, err := resolveDraftID(mailboxAddress, mailboxPath, draftArgs, latest)
			if err != nil {
				return err
			}
			draft, err := mailstore.DetachFileFromDraft(filepath.Join(mailboxPath, "drafts", draftID), attachmentName, time.Now())
			if err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, draftFields(*draft))
			return nil
		},
	}
	cmd.Flags().StringVar(&mailboxAddress, "mailbox", "", "synced mailbox address, e.g. hello@example.com")
	cmd.Flags().BoolVar(&latest, "latest", false, "detach from the newest local draft")
	_ = cmd.MarkFlagRequired("mailbox")
	return cmd
}

func newDraftDeleteCommand(rt *runtime) *cobra.Command {
	var mailboxAddress string
	var latest bool
	cmd := &cobra.Command{
		Use:   "delete [draft-id]",
		Short: "Delete a local or synced remote draft",
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
			draftPath := filepath.Join(mailboxPath, "drafts", draftID)
			draft, err := mailstore.ReadDraft(draftPath)
			if err != nil {
				return err
			}
			if draft.Meta.RemoteID > 0 {
				service, err := mailService(rt)
				if err != nil {
					return err
				}
				if err := service.DeleteOutboundMessage(rt.context(), draft.Meta.RemoteID); err != nil {
					return err
				}
			}
			if err := mailstore.DeleteDraft(draftPath); err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, [][]string{{"deleted", draft.Meta.ID}})
			return nil
		},
	}
	cmd.Flags().StringVar(&mailboxAddress, "mailbox", "", "synced mailbox address, e.g. hello@example.com")
	cmd.Flags().BoolVar(&latest, "latest", false, "delete the newest local draft")
	_ = cmd.MarkFlagRequired("mailbox")
	return cmd
}

func newDraftSendCommand(rt *runtime) *cobra.Command {
	var mailboxAddress string
	var latest bool
	cmd := &cobra.Command{
		Use:   "send [draft-id]",
		Short: "Send a local draft and move it to outbox",
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
				if errors.Is(err, os.ErrNotExist) {
					return draftNotFoundError(mailboxAddress, draftID, mailboxPath)
				}
				return err
			}
			service, err := mailService(rt)
			if err != nil {
				return err
			}
			sent, err := mailsend.SendDraft(rt.context(), store, service, *mailbox, *draft)
			if err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, [][]string{
				{"draft_id", sent.DraftID},
				{"remote_id", strconv.FormatInt(sent.RemoteID, 10)},
				{"status", sent.Status},
				{"path", sent.Path},
			})
			return nil
		},
	}
	cmd.Flags().StringVar(&mailboxAddress, "mailbox", "", "synced mailbox address, e.g. hello@example.com")
	cmd.Flags().BoolVar(&latest, "latest", false, "send the newest local draft")
	_ = cmd.MarkFlagRequired("mailbox")
	return cmd
}

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

func resolveDraftID(mailboxAddress, mailboxPath string, args []string, latest bool) (string, error) {
	if len(args) > 0 && latest {
		return "", fmt.Errorf("provide either a draft ID or --latest, not both")
	}
	if len(args) > 0 {
		return args[0], nil
	}
	drafts, err := mailstore.ListDrafts(mailboxPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("no local drafts exist for %s", mailboxAddress)
		}
		return "", err
	}
	if len(drafts) == 0 {
		return "", fmt.Errorf("no local drafts exist for %s", mailboxAddress)
	}
	if latest || len(drafts) == 1 {
		return drafts[0].Meta.ID, nil
	}
	ids := make([]string, 0, len(drafts))
	for _, draft := range drafts {
		ids = append(ids, draft.Meta.ID)
	}
	return "", fmt.Errorf("multiple drafts exist for %s; provide a draft ID or use --latest. Available drafts: %s", mailboxAddress, strings.Join(ids, ", "))
}

func draftNotFoundError(mailboxAddress, draftID, mailboxPath string) error {
	drafts, err := mailstore.ListDrafts(mailboxPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("draft %q was not found for %s; no local drafts exist yet", draftID, mailboxAddress)
		}
		return fmt.Errorf("draft %q was not found for %s; listing drafts failed: %w", draftID, mailboxAddress, err)
	}
	if len(drafts) == 0 {
		return fmt.Errorf("draft %q was not found for %s; no local drafts exist yet", draftID, mailboxAddress)
	}
	ids := make([]string, 0, len(drafts))
	for _, draft := range drafts {
		ids = append(ids, draft.Meta.ID)
	}
	return fmt.Errorf("draft %q was not found for %s; available drafts: %s", draftID, mailboxAddress, strings.Join(ids, ", "))
}

func draftRows(drafts []mailstore.Draft) [][]string {
	rows := make([][]string, 0, len(drafts))
	for _, draft := range drafts {
		rows = append(rows, []string{
			strconv.FormatInt(draft.Meta.RemoteID, 10),
			draft.Meta.RemoteStatus,
			draft.Meta.Subject,
			strings.Join(draft.Meta.To, ", "),
			draft.Meta.UpdatedAt.Format("2006-01-02 15:04"),
			draft.Path,
		})
	}
	return rows
}

func draftFields(draft mailstore.Draft) [][]string {
	rows := [][]string{
		{"id", draft.Meta.ID},
		{"from", draft.Meta.FromAddress},
		{"to", strings.Join(draft.Meta.To, ", ")},
		{"cc", strings.Join(draft.Meta.CC, ", ")},
		{"bcc", strings.Join(draft.Meta.BCC, ", ")},
		{"subject", draft.Meta.Subject},
		{"updated_at", draft.Meta.UpdatedAt.Format("2006-01-02 15:04")},
		{"attachments", strconv.Itoa(len(draft.Meta.Attachments))},
		{"path", draft.Path},
	}
	if draft.Meta.SourceMessageID > 0 {
		rows = append(rows, []string{"source_message_id", strconv.FormatInt(draft.Meta.SourceMessageID, 10)})
	}
	if draft.Meta.RemoteID > 0 {
		rows = append(rows, []string{"remote_id", strconv.FormatInt(draft.Meta.RemoteID, 10)})
	}
	if draft.Meta.ConversationID > 0 {
		rows = append(rows, []string{"conversation_id", strconv.FormatInt(draft.Meta.ConversationID, 10)})
	}
	return rows
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

func outboundInputFromDraft(draft mailstore.Draft) *mail.OutboundMessageInput {
	domainID := draft.Meta.DomainID
	inboxID := draft.Meta.InboxID
	return &mail.OutboundMessageInput{
		DomainID:        &domainID,
		InboxID:         &inboxID,
		SourceMessageID: int64Ptr(draft.Meta.SourceMessageID),
		ConversationID:  int64Ptr(draft.Meta.ConversationID),
		ToAddresses:     draft.Meta.To,
		CCAddresses:     draft.Meta.CC,
		BCCAddresses:    draft.Meta.BCC,
		Subject:         draft.Meta.Subject,
		Body:            draft.Body,
	}
}

func int64Ptr(value int64) *int64 {
	if value == 0 {
		return nil
	}
	return &value
}

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

func newMessagesCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "messages", Short: "Inspect and triage inbound messages"}
	cmd.AddCommand(newMessagesListCommand(rt))
	cmd.AddCommand(newMessagesShowCommand(rt))
	cmd.AddCommand(newMessagesBodyCommand(rt))
	cmd.AddCommand(newMessageLabelsCommand(rt))
	cmd.AddCommand(newMessageActionCommand(rt, "archive", "Archive a message", func(s *mail.Service, id int64) (*mail.Message, error) { return s.ArchiveMessage(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "restore", "Restore a message", func(s *mail.Service, id int64) (*mail.Message, error) { return s.RestoreMessage(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "trash", "Move a message to trash", func(s *mail.Service, id int64) (*mail.Message, error) { return s.TrashMessage(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "mark-read", "Mark a message read", func(s *mail.Service, id int64) (*mail.Message, error) { return s.MarkMessageRead(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "mark-unread", "Mark a message unread", func(s *mail.Service, id int64) (*mail.Message, error) { return s.MarkMessageUnread(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "star", "Star a message", func(s *mail.Service, id int64) (*mail.Message, error) { return s.StarMessage(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "unstar", "Unstar a message", func(s *mail.Service, id int64) (*mail.Message, error) { return s.UnstarMessage(rt.context(), id) }))
	return cmd
}

func newMessageLabelsCommand(rt *runtime) *cobra.Command {
	var add []int64
	var remove []int64
	var set []int64
	cmd := &cobra.Command{
		Use:   "labels <id>",
		Short: "Show or update message labels",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			service, err := mailService(rt)
			if err != nil {
				return err
			}
			message, err := service.ShowMessage(rt.context(), id)
			if err != nil {
				return err
			}
			if len(add) > 0 || len(remove) > 0 || len(set) > 0 {
				ids := set
				if len(set) == 0 {
					ids = updatedLabelIDs(message.Labels, add, remove)
				}
				message, err = service.AssignMessageLabels(rt.context(), id, ids)
				if err != nil {
					return err
				}
			}
			rows := make([][]string, 0, len(message.Labels))
			for _, label := range message.Labels {
				rows = append(rows, labelRow(label))
			}
			writeRows(cmd.OutOrStdout(), []string{"id", "name", "color"}, rows)
			return nil
		},
	}
	cmd.Flags().Int64SliceVar(&add, "add", nil, "label ID to add, repeatable or comma-separated")
	cmd.Flags().Int64SliceVar(&remove, "remove", nil, "label ID to remove, repeatable or comma-separated")
	cmd.Flags().Int64SliceVar(&set, "set", nil, "replace labels with these IDs, repeatable or comma-separated")
	return cmd
}

func newConversationsCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "conversations", Short: "Inspect conversation threads"}
	cmd.AddCommand(newConversationTimelineCommand(rt))
	return cmd
}

func newConversationTimelineCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "timeline <id>",
		Short: "Show a remote conversation timeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			service, err := mailService(rt)
			if err != nil {
				return err
			}
			entries, err := service.ConversationTimeline(rt.context(), id)
			if err != nil {
				return err
			}
			rows := make([][]string, 0, len(entries))
			for _, entry := range entries {
				rows = append(rows, conversationTimelineRow(entry))
			}
			writeRows(cmd.OutOrStdout(), []string{"kind", "id", "subject", "sender", "recipients", "status", "occurred_at"}, rows)
			return nil
		},
	}
}

func newMessagesListCommand(rt *runtime) *cobra.Command {
	var params mail.MessageListParams
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMessageList(cmd, rt, params)
		},
	}
	addMessageListFlags(cmd, &params)
	return cmd
}

func addMessageListFlags(cmd *cobra.Command, params *mail.MessageListParams) {
	cmd.Flags().IntVar(&params.Page, "page", 1, "page number")
	cmd.Flags().IntVar(&params.PerPage, "per-page", 25, "items per page")
	cmd.Flags().Int64Var(&params.InboxID, "inbox-id", 0, "filter by inbox ID")
	cmd.Flags().StringVar(&params.Mailbox, "mailbox", "", "filter by mailbox: inbox, junk, archived, trash")
	cmd.Flags().Int64Var(&params.LabelID, "label-id", 0, "filter by label ID")
	cmd.Flags().StringVarP(&params.Query, "query", "q", "", "search query")
	cmd.Flags().StringVar(&params.Sender, "sender", "", "filter by sender name or address")
	cmd.Flags().StringVar(&params.Recipient, "recipient", "", "filter by recipient address")
	cmd.Flags().StringVar(&params.Status, "status", "", "filter by processing status")
	cmd.Flags().StringVar(&params.Subaddress, "subaddress", "", "filter by inbox subaddress")
	cmd.Flags().StringVar(&params.ReceivedFrom, "received-from", "", "filter by received date from YYYY-MM-DD")
	cmd.Flags().StringVar(&params.ReceivedTo, "received-to", "", "filter by received date to YYYY-MM-DD")
	cmd.Flags().StringVar(&params.Sort, "sort", "-received_at", "sort order")
}

func runMessageList(cmd *cobra.Command, rt *runtime, params mail.MessageListParams) error {
	service, err := mailService(rt)
	if err != nil {
		return err
	}
	messages, _, err := service.ListMessages(rt.context(), params)
	if err != nil {
		return err
	}
	rows := make([][]string, 0, len(messages))
	for _, message := range messages {
		rows = append(rows, messageRow(message))
	}
	writeRows(cmd.OutOrStdout(), []string{"id", "subject", "from", "status", "mailbox", "read", "starred", "received_at"}, rows)
	return nil
}

func newMessagesShowCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a message",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			service, err := mailService(rt)
			if err != nil {
				return err
			}
			message, err := service.ShowMessage(rt.context(), id)
			if err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, messageFields(*message))
			return nil
		},
	}
}

func newMessagesBodyCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "body <id>",
		Short: "Show message body text or HTML",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			service, err := mailService(rt)
			if err != nil {
				return err
			}
			body, err := service.MessageBody(rt.context(), id)
			if err != nil {
				return err
			}
			content := strings.TrimSpace(body.Text)
			if content == "" {
				content = strings.TrimSpace(body.HTML)
			}
			fmt.Fprintln(cmd.OutOrStdout(), content)
			return nil
		},
	}
}

func newMessageActionCommand(rt *runtime, use, short string, run func(*mail.Service, int64) (*mail.Message, error)) *cobra.Command {
	return &cobra.Command{
		Use:   use + " <id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			service, err := mailService(rt)
			if err != nil {
				return err
			}
			message, err := run(service, id)
			if err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, messageFields(*message))
			return nil
		},
	}
}

func mailService(rt *runtime) (*mail.Service, error) {
	client, err := rt.apiClient()
	if err != nil {
		return nil, err
	}
	return mail.NewService(client), nil
}

func parseID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid id %q", value)
	}
	return id, nil
}

func messageRow(message mail.Message) []string {
	return []string{
		strconv.FormatInt(message.ID, 10),
		message.Subject,
		message.FromAddress,
		message.Status,
		message.SystemState,
		strconv.FormatBool(message.Read),
		strconv.FormatBool(message.Starred),
		message.ReceivedAt.Format("2006-01-02 15:04"),
	}
}

func messageFields(message mail.Message) [][]string {
	return [][]string{
		{"id", strconv.FormatInt(message.ID, 10)},
		{"subject", message.Subject},
		{"from", message.FromAddress},
		{"to", strings.Join(message.ToAddresses, ", ")},
		{"cc", strings.Join(message.CCAddresses, ", ")},
		{"status", message.Status},
		{"mailbox", message.SystemState},
		{"read", strconv.FormatBool(message.Read)},
		{"starred", strconv.FormatBool(message.Starred)},
		{"received_at", message.ReceivedAt.Format("2006-01-02 15:04")},
		{"preview", message.PreviewText},
	}
}

func conversationTimelineRow(entry mail.ConversationTimelineEntry) []string {
	return []string{
		entry.Kind,
		strconv.FormatInt(entry.RecordID, 10),
		entry.Subject,
		entry.Sender,
		strings.Join(entry.Recipients, ", "),
		entry.Status,
		entry.OccurredAt.Format("2006-01-02 15:04"),
	}
}

func labelRow(label mail.Label) []string {
	return []string{strconv.FormatInt(label.ID, 10), label.Name, label.Color}
}

func updatedLabelIDs(current []mail.Label, add, remove []int64) []int64 {
	ids := make(map[int64]bool, len(current)+len(add))
	for _, label := range current {
		ids[label.ID] = true
	}
	for _, id := range add {
		if id > 0 {
			ids[id] = true
		}
	}
	for _, id := range remove {
		delete(ids, id)
	}
	out := make([]int64, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

func cachedLabelNames(labels []mailstore.LabelMeta) []string {
	names := make([]string, 0, len(labels))
	for _, label := range labels {
		if strings.TrimSpace(label.Name) != "" {
			names = append(names, label.Name)
		}
	}
	return names
}

func cachedMessageRow(message mailstore.CachedMessage) []string {
	return []string{
		strconv.FormatInt(message.Meta.RemoteID, 10),
		message.Meta.FromAddress,
		message.Meta.Subject,
		strconv.FormatBool(message.Meta.Read),
		strconv.FormatBool(message.Meta.Starred),
		message.Meta.ReceivedAt.Format("2006-01-02 15:04"),
		message.Path,
	}
}

func cachedMessageFields(message mailstore.CachedMessage) [][]string {
	return [][]string{
		{"id", strconv.FormatInt(message.Meta.RemoteID, 10)},
		{"subject", message.Meta.Subject},
		{"from", message.Meta.FromAddress},
		{"from_name", message.Meta.FromName},
		{"to", strings.Join(message.Meta.To, ", ")},
		{"cc", strings.Join(message.Meta.CC, ", ")},
		{"mailbox", message.Meta.Mailbox},
		{"read", strconv.FormatBool(message.Meta.Read)},
		{"starred", strconv.FormatBool(message.Meta.Starred)},
		{"labels", strings.Join(cachedLabelNames(message.Meta.Labels), ", ")},
		{"received_at", message.Meta.ReceivedAt.Format("2006-01-02 15:04")},
		{"path", message.Path},
	}
}

func inboxAddresses(inboxes []mail.Inbox) []string {
	values := make([]string, 0, len(inboxes))
	for _, inbox := range inboxes {
		values = append(values, inbox.Address)
	}
	return values
}

func domainNames(domains []mail.Domain) []string {
	values := make([]string, 0, len(domains))
	for _, domain := range domains {
		values = append(values, domain.Name)
	}
	return values
}

func splitAddresses(values []string) []string {
	addresses := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				addresses = append(addresses, part)
			}
		}
	}
	return addresses
}
