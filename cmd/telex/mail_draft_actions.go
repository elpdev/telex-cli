package main

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/elpdev/telex-cli/internal/api"
	"github.com/elpdev/telex-cli/internal/mailsend"
	"github.com/elpdev/telex-cli/internal/mailstore"
	"github.com/spf13/cobra"
)

func newDraftAttachCommand(rt *runtime) *cobra.Command {
	var mailboxAddress string
	var latest bool
	cmd := &cobra.Command{
		Use:   "attach [draft-id] <file>",
		Short: "Attach a local file to a draft",
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
	cmd.Flags().BoolVar(&latest, "latest", false, "attach to the newest draft")
	_ = cmd.MarkFlagRequired("mailbox")
	return cmd
}

func newDraftDetachCommand(rt *runtime) *cobra.Command {
	var mailboxAddress string
	var latest bool
	cmd := &cobra.Command{
		Use:   "detach [draft-id] <attachment>",
		Short: "Detach a local file from a draft",
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
	cmd.Flags().BoolVar(&latest, "latest", false, "detach from the newest draft")
	_ = cmd.MarkFlagRequired("mailbox")
	return cmd
}

func newDraftDeleteCommand(rt *runtime) *cobra.Command {
	var mailboxAddress string
	var latest bool
	cmd := &cobra.Command{
		Use:   "delete [draft-id]",
		Short: "Delete a draft",
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
			if mailstore.HasRemoteDraft(*draft) {
				service, err := mailService(rt)
				if err != nil {
					return err
				}
				if err := service.DeleteOutboundMessage(rt.context(), draft.Meta.RemoteID); err != nil && !api.IsStatus(err, http.StatusNotFound) {
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
	cmd.Flags().BoolVar(&latest, "latest", false, "delete the newest draft")
	_ = cmd.MarkFlagRequired("mailbox")
	return cmd
}

func newDraftSendCommand(rt *runtime) *cobra.Command {
	var mailboxAddress string
	var latest bool
	cmd := &cobra.Command{
		Use:   "send [draft-id]",
		Short: "Send a draft and move it to outbox",
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
	cmd.Flags().BoolVar(&latest, "latest", false, "send the newest draft")
	_ = cmd.MarkFlagRequired("mailbox")
	return cmd
}
