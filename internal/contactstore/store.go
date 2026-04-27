package contactstore

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/elpdev/telex-cli/internal/contacts"
	"github.com/elpdev/telex-cli/internal/mailstore"
)

const SchemaVersion = 1

type Store struct {
	Root string
}

type StoreMeta struct {
	SchemaVersion int       `toml:"schema_version"`
	SyncedAt      time.Time `toml:"synced_at"`
}

type ContactMeta struct {
	SchemaVersion       int                       `toml:"schema_version"`
	RemoteID            int64                     `toml:"remote_id"`
	UserID              int64                     `toml:"user_id"`
	ContactType         string                    `toml:"contact_type"`
	Name                string                    `toml:"name"`
	CompanyName         string                    `toml:"company_name"`
	Title               string                    `toml:"title"`
	Phone               string                    `toml:"phone"`
	Website             string                    `toml:"website"`
	DisplayName         string                    `toml:"display_name"`
	PrimaryEmailAddress string                    `toml:"primary_email_address"`
	EmailAddresses      []ContactEmailAddressMeta `toml:"email_addresses"`
	NoteFileID          int64                     `toml:"note_file_id"`
	Metadata            map[string]any            `toml:"metadata"`
	RemoteCreatedAt     time.Time                 `toml:"remote_created_at"`
	RemoteUpdatedAt     time.Time                 `toml:"remote_updated_at"`
	SyncedAt            time.Time                 `toml:"synced_at"`
}

type ContactEmailAddressMeta struct {
	ID             int64     `toml:"id"`
	EmailAddress   string    `toml:"email_address"`
	Label          string    `toml:"label"`
	PrimaryAddress bool      `toml:"primary_address"`
	CreatedAt      time.Time `toml:"created_at"`
	UpdatedAt      time.Time `toml:"updated_at"`
}

type ContactNoteMeta struct {
	ContactID    int64      `toml:"contact_id"`
	StoredFileID int64      `toml:"stored_file_id"`
	Title        string     `toml:"title"`
	CreatedAt    *time.Time `toml:"created_at"`
	UpdatedAt    *time.Time `toml:"updated_at"`
	SyncedAt     time.Time  `toml:"synced_at"`
}

type CommunicationMeta struct {
	ID               int64          `toml:"id"`
	ContactID        int64          `toml:"contact_id"`
	Kind             string         `toml:"kind"`
	CommunicableType string         `toml:"communicable_type"`
	CommunicableID   int64          `toml:"communicable_id"`
	OccurredAt       time.Time      `toml:"occurred_at"`
	Metadata         map[string]any `toml:"metadata"`
	Subject          string         `toml:"subject"`
	PreviewText      string         `toml:"preview_text"`
	FromAddress      string         `toml:"from_address"`
	SenderDisplay    string         `toml:"sender_display"`
	Status           string         `toml:"status"`
	Direction        string         `toml:"direction"`
	RemoteCreatedAt  time.Time      `toml:"remote_created_at"`
	RemoteUpdatedAt  time.Time      `toml:"remote_updated_at"`
}

type CachedContact struct {
	Meta           ContactMeta
	Note           *CachedContactNote
	Communications []CommunicationMeta
	Path           string
}

type CachedContactNote struct {
	Meta ContactNoteMeta
	Body string
}

func New(root string) Store {
	return Store{Root: mailstore.RootOrDefault(root)}
}

func (s Store) ContactsRoot() string { return filepath.Join(s.Root, "contacts") }

func (s Store) EnsureRoot() error {
	for _, dir := range []string{s.ContactsRoot(), s.recordsRoot()} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) StoreSyncMeta(syncedAt time.Time) error {
	if err := s.EnsureRoot(); err != nil {
		return err
	}
	return writeTOML(filepath.Join(s.ContactsRoot(), "meta.toml"), StoreMeta{SchemaVersion: SchemaVersion, SyncedAt: syncedAt})
}

func (s Store) StoreContact(contact contacts.Contact, syncedAt time.Time) error {
	if err := s.EnsureRoot(); err != nil {
		return err
	}
	path := s.contactPath(contact.ID)
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	meta := contactMeta(contact, syncedAt)
	if err := writeTOML(filepath.Join(path, "meta.toml"), meta); err != nil {
		return err
	}
	if contact.Note != nil {
		return s.StoreContactNote(*contact.Note, syncedAt)
	}
	return nil
}

func (s Store) StoreContactNote(note contacts.ContactNote, syncedAt time.Time) error {
	if err := s.EnsureRoot(); err != nil {
		return err
	}
	path := s.contactPath(note.ContactID)
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	meta := ContactNoteMeta{ContactID: note.ContactID, Title: note.Title, CreatedAt: note.CreatedAt, UpdatedAt: note.UpdatedAt, SyncedAt: syncedAt}
	if note.StoredFileID != nil {
		meta.StoredFileID = *note.StoredFileID
	}
	if err := writeTOML(filepath.Join(path, "note.toml"), meta); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(path, "note.md"), []byte(note.Body), 0o600)
}

func (s Store) StoreCommunications(contactID int64, communications []contacts.ContactCommunication) error {
	if err := s.EnsureRoot(); err != nil {
		return err
	}
	path := s.contactPath(contactID)
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	metas := make([]CommunicationMeta, 0, len(communications))
	for _, communication := range communications {
		metas = append(metas, communicationMeta(communication))
	}
	return writeTOML(filepath.Join(path, "communications.toml"), struct {
		Communications []CommunicationMeta `toml:"communications"`
	}{Communications: metas})
}

func (s Store) ListContacts() ([]CachedContact, error) {
	entries, err := os.ReadDir(s.recordsRoot())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := []CachedContact{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		cached, err := s.ReadContactPath(filepath.Join(s.recordsRoot(), entry.Name()))
		if err != nil {
			continue
		}
		out = append(out, *cached)
	}
	sort.Slice(out, func(i, j int) bool {
		left := strings.ToLower(out[i].Meta.DisplayName)
		right := strings.ToLower(out[j].Meta.DisplayName)
		if left == right {
			return out[i].Meta.RemoteID < out[j].Meta.RemoteID
		}
		return left < right
	})
	return out, nil
}

func (s Store) ReadContact(id int64) (*CachedContact, error) {
	return s.ReadContactPath(s.contactPath(id))
}

func (s Store) ReadContactPath(path string) (*CachedContact, error) {
	var meta ContactMeta
	if _, err := toml.DecodeFile(filepath.Join(path, "meta.toml"), &meta); err != nil {
		return nil, err
	}
	cached := &CachedContact{Meta: meta, Path: path}
	if note, err := readNote(path); err == nil {
		cached.Note = note
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	communications, err := readCommunications(path)
	if err != nil {
		return nil, err
	}
	cached.Communications = communications
	return cached, nil
}

func (s Store) DeleteContact(id int64) error {
	err := os.RemoveAll(s.contactPath(id))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s Store) Counts() (int, error) {
	contacts, err := s.ListContacts()
	if err != nil {
		return 0, err
	}
	return len(contacts), nil
}

func readNote(path string) (*CachedContactNote, error) {
	var meta ContactNoteMeta
	if _, err := toml.DecodeFile(filepath.Join(path, "note.toml"), &meta); err != nil {
		return nil, err
	}
	body, err := os.ReadFile(filepath.Join(path, "note.md"))
	if err != nil {
		return nil, err
	}
	return &CachedContactNote{Meta: meta, Body: string(body)}, nil
}

func readCommunications(path string) ([]CommunicationMeta, error) {
	var payload struct {
		Communications []CommunicationMeta `toml:"communications"`
	}
	if _, err := toml.DecodeFile(filepath.Join(path, "communications.toml"), &payload); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return payload.Communications, nil
}

func (s Store) recordsRoot() string { return filepath.Join(s.ContactsRoot(), "records") }

func (s Store) contactPath(id int64) string {
	return filepath.Join(s.recordsRoot(), fmt.Sprintf("%d", id))
}

func contactMeta(contact contacts.Contact, syncedAt time.Time) ContactMeta {
	meta := ContactMeta{SchemaVersion: SchemaVersion, RemoteID: contact.ID, UserID: contact.UserID, ContactType: contact.ContactType, Name: contact.Name, CompanyName: contact.CompanyName, Title: contact.Title, Phone: contact.Phone, Website: contact.Website, DisplayName: contact.DisplayName, PrimaryEmailAddress: contact.PrimaryEmailAddress, EmailAddresses: emailAddressMetas(contact.EmailAddresses), Metadata: contact.Metadata, RemoteCreatedAt: contact.CreatedAt, RemoteUpdatedAt: contact.UpdatedAt, SyncedAt: syncedAt}
	if contact.NoteFileID != nil {
		meta.NoteFileID = *contact.NoteFileID
	}
	return meta
}

func emailAddressMetas(values []contacts.ContactEmailAddress) []ContactEmailAddressMeta {
	out := make([]ContactEmailAddressMeta, 0, len(values))
	for _, value := range values {
		out = append(out, ContactEmailAddressMeta{ID: value.ID, EmailAddress: value.EmailAddress, Label: value.Label, PrimaryAddress: value.PrimaryAddress, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt})
	}
	return out
}

func communicationMeta(value contacts.ContactCommunication) CommunicationMeta {
	meta := CommunicationMeta{ID: value.ID, ContactID: value.ContactID, Kind: value.Kind, CommunicableType: value.CommunicableType, CommunicableID: value.CommunicableID, OccurredAt: value.OccurredAt, Metadata: value.Metadata, RemoteCreatedAt: value.CreatedAt, RemoteUpdatedAt: value.UpdatedAt}
	meta.Subject = stringFromAny(value.Communication["subject"])
	meta.PreviewText = stringFromAny(value.Communication["preview_text"])
	meta.FromAddress = stringFromAny(value.Communication["from_address"])
	meta.SenderDisplay = stringFromAny(value.Communication["sender_display"])
	meta.Status = stringFromAny(value.Communication["status"])
	meta.Direction = stringFromAny(value.Metadata["direction"])
	return meta
}

func stringFromAny(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func writeTOML(path string, value any) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer closeSilently(file)
	return toml.NewEncoder(file).Encode(value)
}

func closeSilently(file *os.File) {
	_ = file.Close()
}
