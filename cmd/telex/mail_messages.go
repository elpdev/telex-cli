package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/elpdev/telex-cli/internal/mail"
	"github.com/elpdev/telex-cli/internal/mailstore"
	"github.com/spf13/cobra"
)

func newMessagesCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "messages", Short: "Inspect and triage inbound messages"}
	cmd.AddCommand(newMessagesListCommand(rt))
	cmd.AddCommand(newMessagesShowCommand(rt))
	cmd.AddCommand(newMessagesBodyCommand(rt))
	cmd.AddCommand(newMessageLabelsCommand(rt))
	cmd.AddCommand(newMessageActionCommand(rt, "archive", "Archive a message", func(s *mail.Service, id int64) (*mail.Message, error) { return s.ArchiveMessage(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "restore", "Restore a message", func(s *mail.Service, id int64) (*mail.Message, error) { return s.RestoreMessage(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "trash", "Move a message to trash", func(s *mail.Service, id int64) (*mail.Message, error) { return s.TrashMessage(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "junk", "Move a message to junk", func(s *mail.Service, id int64) (*mail.Message, error) { return s.JunkMessage(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "not-junk", "Move a message out of junk", func(s *mail.Service, id int64) (*mail.Message, error) { return s.NotJunkMessage(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "mark-read", "Mark a message read", func(s *mail.Service, id int64) (*mail.Message, error) { return s.MarkMessageRead(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "mark-unread", "Mark a message unread", func(s *mail.Service, id int64) (*mail.Message, error) { return s.MarkMessageUnread(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "star", "Star a message", func(s *mail.Service, id int64) (*mail.Message, error) { return s.StarMessage(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "unstar", "Unstar a message", func(s *mail.Service, id int64) (*mail.Message, error) { return s.UnstarMessage(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "block-sender", "Block the message sender", func(s *mail.Service, id int64) (*mail.Message, error) { return s.BlockSender(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "unblock-sender", "Unblock the message sender", func(s *mail.Service, id int64) (*mail.Message, error) { return s.UnblockSender(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "block-domain", "Block the sender domain", func(s *mail.Service, id int64) (*mail.Message, error) { return s.BlockDomain(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "unblock-domain", "Unblock the sender domain", func(s *mail.Service, id int64) (*mail.Message, error) { return s.UnblockDomain(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "trust-sender", "Trust the message sender", func(s *mail.Service, id int64) (*mail.Message, error) { return s.TrustSender(rt.context(), id) }))
	cmd.AddCommand(newMessageActionCommand(rt, "untrust-sender", "Untrust the message sender", func(s *mail.Service, id int64) (*mail.Message, error) { return s.UntrustSender(rt.context(), id) }))
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
				_, _ = mailstore.New(rt.dataPath).UpdateCachedMessageByRemoteID(id, *message, time.Now())
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
	writeRows(cmd.OutOrStdout(), []string{"id", "subject", "from", "status", "mailbox", "read", "starred", "labels", "received_at"}, rows)
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
			_, _ = mailstore.New(rt.dataPath).UpdateCachedMessageByRemoteID(id, *message, time.Now())
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, messageFields(*message))
			return nil
		},
	}
}
