package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/elpdev/telex-cli/internal/contacts"
	"github.com/elpdev/telex-cli/internal/contactstore"
	"github.com/elpdev/telex-cli/internal/contactsync"
	"github.com/spf13/cobra"
)

type contactsSyncResult = contactsync.Result

func newContactsCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "contacts", Short: "Contacts commands"}
	cmd.AddCommand(newContactsSyncCommand(rt))
	cmd.AddCommand(newContactsListCommand(rt))
	cmd.AddCommand(newContactsShowCommand(rt))
	cmd.AddCommand(newContactsCreateCommand(rt))
	cmd.AddCommand(newContactsEditCommand(rt))
	cmd.AddCommand(newContactsDeleteCommand(rt))
	cmd.AddCommand(newContactsImportVCFCommand(rt))
	cmd.AddCommand(newContactNoteCommand(rt))
	cmd.AddCommand(newContactCommunicationsCommand(rt))
	return cmd
}

func newContactsSyncCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync remote contacts into the local cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := contactsService(rt)
			if err != nil {
				return err
			}
			result, err := runContactsSync(rt, service, contactstore.New(rt.dataPath))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Synced %d contact(s), %d note(s).\n", result.Contacts, result.Notes)
			return nil
		},
	}
}

func newContactsListCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List cached contacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			cached, err := contactstore.New(rt.dataPath).ListContacts()
			if err != nil {
				return err
			}
			rows := make([][]string, 0, len(cached))
			for _, contact := range cached {
				rows = append(rows, cachedContactRow(contact))
			}
			writeRows(cmd.OutOrStdout(), []string{"id", "type", "name", "email", "company", "title", "updated_at", "path"}, rows)
			return nil
		},
	}
}

func newContactsShowCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a cached contact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			cached, err := contactstore.New(rt.dataPath).ReadContact(id)
			if err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, cachedContactFields(*cached))
			if cached.Note != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", cached.Note.Body)
			}
			return nil
		},
	}
}

func newContactsCreateCommand(rt *runtime) *cobra.Command {
	var input contacts.ContactInput
	var email string
	var emailLabel string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a remote contact and cache it locally",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(input.Name) == "" && strings.TrimSpace(input.CompanyName) == "" && strings.TrimSpace(email) == "" {
				return fmt.Errorf("provide --name, --company, or --email")
			}
			if input.ContactType == "" {
				input.ContactType = "person"
			}
			if email != "" {
				primary := true
				input.EmailAddresses = []contacts.ContactEmailAddressInput{{EmailAddress: email, Label: emailLabel, PrimaryAddress: &primary}}
			}
			service, err := contactsService(rt)
			if err != nil {
				return err
			}
			contact, err := service.CreateContact(rt.context(), input)
			if err != nil {
				return err
			}
			if err := contactstore.New(rt.dataPath).StoreContact(*contact, time.Now()); err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, contactFields(*contact))
			return nil
		},
	}
	addContactInputFlags(cmd, &input)
	cmd.Flags().StringVar(&email, "email", "", "primary email address")
	cmd.Flags().StringVar(&emailLabel, "email-label", "email", "primary email label")
	return cmd
}

func newContactsEditCommand(rt *runtime) *cobra.Command {
	var input contacts.ContactInput
	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Update a remote contact and refresh the local cache",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			service, err := contactsService(rt)
			if err != nil {
				return err
			}
			updated, err := service.UpdateContact(rt.context(), id, input)
			if err != nil {
				return err
			}
			if err := contactstore.New(rt.dataPath).StoreContact(*updated, time.Now()); err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, contactFields(*updated))
			return nil
		},
	}
	addContactInputFlags(cmd, &input)
	return cmd
}

func newContactsDeleteCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a remote contact and remove the local cache",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			service, err := contactsService(rt)
			if err != nil {
				return err
			}
			if err := service.DeleteContact(rt.context(), id); err != nil {
				return err
			}
			if err := contactstore.New(rt.dataPath).DeleteContact(id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted contact %d.\n", id)
			return nil
		},
	}
}

func newContactNoteCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "note", Short: "Contact note commands"}
	cmd.AddCommand(newContactNoteShowCommand(rt))
	cmd.AddCommand(newContactNoteSetCommand(rt))
	return cmd
}

func newContactNoteShowCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a remote contact note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			service, err := contactsService(rt)
			if err != nil {
				return err
			}
			note, err := service.ContactNote(rt.context(), id)
			if err != nil {
				return err
			}
			if err := contactstore.New(rt.dataPath).StoreContactNote(*note, time.Now()); err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, contactNoteFields(*note))
			fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", note.Body)
			return nil
		},
	}
}

func newContactNoteSetCommand(rt *runtime) *cobra.Command {
	var title string
	var body string
	var filePath string
	cmd := &cobra.Command{
		Use:   "set <id>",
		Short: "Create or update a remote contact note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			body, err := noteBodyFromFlags(body, filePath)
			if err != nil {
				return err
			}
			service, err := contactsService(rt)
			if err != nil {
				return err
			}
			note, err := service.UpdateContactNote(rt.context(), id, contacts.ContactNoteInput{Title: title, Body: body})
			if err != nil {
				return err
			}
			if err := contactstore.New(rt.dataPath).StoreContactNote(*note, time.Now()); err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, contactNoteFields(*note))
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "note title")
	cmd.Flags().StringVar(&body, "body", "", "Markdown body")
	cmd.Flags().StringVar(&filePath, "file", "", "read Markdown body from file")
	return cmd
}

func newContactCommunicationsCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "communications <id>",
		Short: "List remote communications for a contact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			service, err := contactsService(rt)
			if err != nil {
				return err
			}
			communications, _, err := service.ContactCommunications(rt.context(), id, contacts.ListParams{Page: 1, PerPage: 100})
			if err != nil {
				return err
			}
			if err := contactstore.New(rt.dataPath).StoreCommunications(id, communications); err != nil {
				return err
			}
			rows := make([][]string, 0, len(communications))
			for _, communication := range communications {
				rows = append(rows, communicationRow(communication))
			}
			writeRows(cmd.OutOrStdout(), []string{"id", "kind", "direction", "subject", "occurred_at"}, rows)
			return nil
		},
	}
}

func newContactsImportVCFCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "import-vcf <file>",
		Short: "Import contacts from a VCF file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := contactsService(rt)
			if err != nil {
				return err
			}
			result, err := service.ImportVCF(rt.context(), args[0])
			if err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, [][]string{{"created", strconv.Itoa(result.Created)}, {"updated", strconv.Itoa(result.Updated)}, {"skipped", strconv.Itoa(result.Skipped)}, {"failed", strconv.Itoa(result.Failed)}, {"success", strconv.FormatBool(result.Success)}})
			for _, importErr := range result.Errors {
				fmt.Fprintln(cmd.ErrOrStderr(), importErr)
			}
			return nil
		},
	}
}

func addContactInputFlags(cmd *cobra.Command, input *contacts.ContactInput) {
	cmd.Flags().StringVar(&input.ContactType, "type", "", "contact type: person or business")
	cmd.Flags().StringVar(&input.Name, "name", "", "contact name")
	cmd.Flags().StringVar(&input.CompanyName, "company", "", "company name")
	cmd.Flags().StringVar(&input.Title, "title", "", "contact title")
	cmd.Flags().StringVar(&input.Phone, "phone", "", "phone number")
	cmd.Flags().StringVar(&input.Website, "website", "", "website URL")
}

func runContactsSync(rt *runtime, service *contacts.Service, store contactstore.Store) (*contactsSyncResult, error) {
	return contactsync.Run(rt.context(), store, service)
}

func contactsService(rt *runtime) (*contacts.Service, error) {
	client, err := rt.apiClient()
	if err != nil {
		return nil, err
	}
	return contacts.NewService(client), nil
}

func cachedContactRow(contact contactstore.CachedContact) []string {
	return []string{strconv.FormatInt(contact.Meta.RemoteID, 10), contact.Meta.ContactType, contact.Meta.DisplayName, contact.Meta.PrimaryEmailAddress, contact.Meta.CompanyName, contact.Meta.Title, contact.Meta.RemoteUpdatedAt.Format("2006-01-02 15:04"), contact.Path}
}

func cachedContactFields(contact contactstore.CachedContact) [][]string {
	emails := make([]string, 0, len(contact.Meta.EmailAddresses))
	for _, email := range contact.Meta.EmailAddresses {
		emails = append(emails, email.EmailAddress)
	}
	return [][]string{{"id", strconv.FormatInt(contact.Meta.RemoteID, 10)}, {"type", contact.Meta.ContactType}, {"name", contact.Meta.Name}, {"display_name", contact.Meta.DisplayName}, {"company", contact.Meta.CompanyName}, {"title", contact.Meta.Title}, {"email", contact.Meta.PrimaryEmailAddress}, {"emails", strings.Join(emails, ", ")}, {"phone", contact.Meta.Phone}, {"website", contact.Meta.Website}, {"updated_at", contact.Meta.RemoteUpdatedAt.Format("2006-01-02 15:04")}, {"path", contact.Path}}
}

func contactFields(contact contacts.Contact) [][]string {
	return [][]string{{"id", strconv.FormatInt(contact.ID, 10)}, {"type", contact.ContactType}, {"display_name", contact.DisplayName}, {"email", contact.PrimaryEmailAddress}, {"company", contact.CompanyName}, {"title", contact.Title}, {"updated_at", contact.UpdatedAt.Format("2006-01-02 15:04")}}
}

func contactNoteFields(note contacts.ContactNote) [][]string {
	storedFileID := ""
	if note.StoredFileID != nil {
		storedFileID = strconv.FormatInt(*note.StoredFileID, 10)
	}
	return [][]string{{"contact_id", strconv.FormatInt(note.ContactID, 10)}, {"stored_file_id", storedFileID}, {"title", note.Title}}
}

func communicationRow(communication contacts.ContactCommunication) []string {
	return []string{strconv.FormatInt(communication.ID, 10), communication.Kind, anyString(communication.Metadata["direction"]), anyString(communication.Communication["subject"]), communication.OccurredAt.Format("2006-01-02 15:04")}
}

func anyString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}
