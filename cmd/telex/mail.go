package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/elpdev/telex-cli/internal/mail"
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
		strings.Join(labelNames(message.Labels), ", "),
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
		{"sender_blocked", strconv.FormatBool(message.SenderBlocked)},
		{"sender_trusted", strconv.FormatBool(message.SenderTrusted)},
		{"domain_blocked", strconv.FormatBool(message.DomainBlocked)},
		{"labels", strings.Join(labelNames(message.Labels), ", ")},
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

func labelNames(labels []mail.Label) []string {
	names := make([]string, 0, len(labels))
	for _, label := range labels {
		if strings.TrimSpace(label.Name) != "" {
			names = append(names, label.Name)
		}
	}
	return names
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
