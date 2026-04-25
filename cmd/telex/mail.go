package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/elpdev/telex-cli/internal/mail"
	"github.com/elpdev/telex-cli/internal/mailstore"
	"github.com/spf13/cobra"
)

func newMailCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "mail", Short: "Email commands"}
	cmd.AddCommand(newMailSyncCommand(rt))
	cmd.AddCommand(newMailboxesCommand(rt))
	cmd.AddCommand(newInboxCommand(rt))
	cmd.AddCommand(newDraftsCommand(rt))
	cmd.AddCommand(newOutboxCommand(rt))
	cmd.AddCommand(newMessagesCommand(rt))
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
	store, result, err := syncMailboxes(rt)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Synced %d active mailbox(es).\n", len(result.Created))
	if len(result.Skipped) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Skipped %d inactive mailbox(es).\n", len(result.Skipped))
	}

	mailboxes := result.Created
	if mailboxAddress != "" {
		mailbox, _, err := store.FindMailboxByAddress(mailboxAddress)
		if err != nil {
			return err
		}
		mailboxes = []mailstore.MailboxMeta{*mailbox}
	}

	rows := [][]string{}
	inboxCount := 0
	bodyErrorCount := 0
	inboxErrorCount := 0
	for _, mailbox := range mailboxes {
		synced, err := syncOutboxForMailbox(rt, store, mailbox)
		if err != nil {
			return fmt.Errorf("sync outbox for %s: %w", mailbox.Address, err)
		}
		for _, row := range synced {
			rows = append(rows, append([]string{mailbox.Address}, row...))
		}
		count, bodyErrors, err := syncInboxForMailbox(rt, store, mailbox)
		if err != nil {
			inboxCount += count
			bodyErrorCount += bodyErrors
			inboxErrorCount++
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: sync inbox for %s: %v\n", mailbox.Address, err)
			continue
		}
		inboxCount += count
		bodyErrorCount += bodyErrors
	}
	if len(rows) > 0 {
		writeRows(cmd.OutOrStdout(), []string{"mailbox", "remote_id", "status", "subject", "path"}, rows)
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "Outbox already synced.")
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Synced %d inbox message(s).\n", inboxCount)
	if bodyErrorCount > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "Skipped %d inbox message body fetch(es) due to remote API errors; metadata was still cached.\n", bodyErrorCount)
	}
	if inboxErrorCount > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "Skipped partial inbox sync for %d mailbox(es) due to remote API errors.\n", inboxErrorCount)
	}
	return nil
}

func newInboxCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "inbox", Short: "Read cached inbox messages"}
	cmd.AddCommand(newInboxListCommand(rt))
	cmd.AddCommand(newInboxShowCommand(rt))
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

func newDraftsCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "drafts", Short: "Manage local drafts"}
	cmd.AddCommand(newDraftCreateCommand(rt))
	cmd.AddCommand(newDraftListCommand(rt))
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
			domainID := draft.Meta.DomainID
			inboxID := draft.Meta.InboxID
			outbound, err := service.CreateOutboundMessage(rt.context(), &mail.OutboundMessageInput{
				DomainID:     &domainID,
				InboxID:      &inboxID,
				ToAddresses:  draft.Meta.To,
				CCAddresses:  draft.Meta.CC,
				BCCAddresses: draft.Meta.BCC,
				Subject:      draft.Meta.Subject,
				Body:         draft.Body,
			}, false)
			if err != nil {
				return err
			}
			sent, err := service.SendOutboundMessage(rt.context(), outbound.ID)
			if err != nil {
				return err
			}
			moved, err := store.MoveDraftToOutbox(*mailbox, draft.Meta.ID, sent.ID, sent.Status, time.Now())
			if err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, [][]string{
				{"draft_id", moved.Meta.ID},
				{"remote_id", strconv.FormatInt(sent.ID, 10)},
				{"status", sent.Status},
				{"path", moved.Path},
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
			rows, err := syncOutboxForMailbox(rt, store, *mailbox)
			if err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"remote_id", "status", "subject", "path"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&mailboxAddress, "mailbox", "", "synced mailbox address, e.g. hello@example.com")
	_ = cmd.MarkFlagRequired("mailbox")
	return cmd
}

func syncMailboxes(rt *runtime) (mailstore.Store, *mailstore.SyncResult, error) {
	service, err := mailService(rt)
	if err != nil {
		return mailstore.Store{}, nil, err
	}
	bootstrap, err := service.Mailboxes(rt.context())
	if err != nil {
		return mailstore.Store{}, nil, err
	}
	store := mailstore.New(rt.dataPath)
	result, err := store.SyncMailboxes(bootstrap, time.Now())
	if err != nil {
		return mailstore.Store{}, nil, err
	}
	return store, result, nil
}

func syncOutboxForMailbox(rt *runtime, store mailstore.Store, mailbox mailstore.MailboxMeta) ([][]string, error) {
	_, mailboxPath, err := store.FindMailboxByAddress(mailbox.Address)
	if err != nil {
		return nil, err
	}
	items, err := mailstore.ListOutbox(mailboxPath)
	if err != nil {
		return nil, err
	}
	service, err := mailService(rt)
	if err != nil {
		return nil, err
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		remote, err := service.ShowOutboundMessage(rt.context(), item.Meta.RemoteID)
		if err != nil {
			return nil, fmt.Errorf("fetch outbound %d: %w", item.Meta.RemoteID, err)
		}
		moved, err := store.SyncOutboxItem(mailbox, remote.ID, remote.Status, remote.LastError, outboundOccurredAt(remote))
		if err != nil {
			return nil, fmt.Errorf("store outbound %d status: %w", remote.ID, err)
		}
		rows = append(rows, []string{strconv.FormatInt(remote.ID, 10), remote.Status, remote.Subject, moved.Path})
	}
	return rows, nil
}

func syncInboxForMailbox(rt *runtime, store mailstore.Store, mailbox mailstore.MailboxMeta) (int, int, error) {
	service, err := mailService(rt)
	if err != nil {
		return 0, 0, err
	}
	count := 0
	bodyErrors := 0
	page := 1
	const perPage = 100
	for {
		messages, pagination, err := service.ListMessages(rt.context(), mail.MessageListParams{
			ListParams: mail.ListParams{Page: page, PerPage: perPage},
			InboxID:    mailbox.InboxID,
			Mailbox:    "inbox",
			Sort:       "-received_at",
		})
		if err != nil {
			return count, bodyErrors, fmt.Errorf("list page %d: %w", page, err)
		}
		if len(messages) == 0 {
			return count, bodyErrors, nil
		}
		for _, message := range messages {
			body, err := service.MessageBody(rt.context(), message.ID)
			if err != nil {
				bodyErrors++
			}
			if _, err := store.StoreInboxMessage(mailbox, message, body, time.Now()); err != nil {
				return count, bodyErrors, fmt.Errorf("store message %d: %w", message.ID, err)
			}
			count++
		}
		if pagination == nil || page*pagination.PerPage >= pagination.TotalCount {
			return count, bodyErrors, nil
		}
		page++
	}
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

func outboundOccurredAt(message *mail.OutboundMessage) time.Time {
	if message == nil {
		return time.Now()
	}
	if message.SentAt != nil {
		return *message.SentAt
	}
	if message.FailedAt != nil {
		return *message.FailedAt
	}
	if message.QueuedAt != nil {
		return *message.QueuedAt
	}
	return message.UpdatedAt
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
			store, result, err := syncMailboxes(rt)
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
	cmd.AddCommand(newMessageActionCommand(rt, "archive", "Archive a message", func(s *mail.Service, id int64) (*mail.Message, error) { return s.ArchiveMessage(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "restore", "Restore a message", func(s *mail.Service, id int64) (*mail.Message, error) { return s.RestoreMessage(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "trash", "Move a message to trash", func(s *mail.Service, id int64) (*mail.Message, error) { return s.TrashMessage(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "mark-read", "Mark a message read", func(s *mail.Service, id int64) (*mail.Message, error) { return s.MarkMessageRead(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "mark-unread", "Mark a message unread", func(s *mail.Service, id int64) (*mail.Message, error) { return s.MarkMessageUnread(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "star", "Star a message", func(s *mail.Service, id int64) (*mail.Message, error) { return s.StarMessage(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "unstar", "Unstar a message", func(s *mail.Service, id int64) (*mail.Message, error) { return s.UnstarMessage(rt.context(), id) }))
	return cmd
}

func newMessagesListCommand(rt *runtime) *cobra.Command {
	var params mail.MessageListParams
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List messages",
		RunE: func(cmd *cobra.Command, args []string) error {
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
		},
	}
	cmd.Flags().IntVar(&params.Page, "page", 1, "page number")
	cmd.Flags().IntVar(&params.PerPage, "per-page", 25, "items per page")
	cmd.Flags().Int64Var(&params.InboxID, "inbox-id", 0, "filter by inbox ID")
	cmd.Flags().StringVar(&params.Mailbox, "mailbox", "", "filter by mailbox")
	cmd.Flags().Int64Var(&params.LabelID, "label-id", 0, "filter by label ID")
	cmd.Flags().StringVarP(&params.Query, "query", "q", "", "search query")
	cmd.Flags().StringVar(&params.Sender, "sender", "", "filter by sender")
	cmd.Flags().StringVar(&params.Recipient, "recipient", "", "filter by recipient")
	cmd.Flags().StringVar(&params.Status, "status", "", "filter by status")
	cmd.Flags().StringVar(&params.Sort, "sort", "-received_at", "sort order")
	return cmd
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
