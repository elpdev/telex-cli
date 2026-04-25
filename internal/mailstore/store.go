package mailstore

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/elpdev/telex-cli/internal/mail"
)

const SchemaVersion = 1

var mailboxDirs = []string{"inbox", "drafts", "outbox", "sent", "failed", "archive", "trash"}

type Store struct {
	Root string
}

type MailboxMeta struct {
	SchemaVersion int       `toml:"schema_version"`
	DomainID      int64     `toml:"domain_id"`
	DomainName    string    `toml:"domain_name"`
	InboxID       int64     `toml:"inbox_id"`
	Address       string    `toml:"address"`
	LocalPart     string    `toml:"local_part"`
	Active        bool      `toml:"active"`
	SyncedAt      time.Time `toml:"synced_at"`
}

type SyncResult struct {
	Created []MailboxMeta
	Skipped []mail.Inbox
}

func DefaultRoot() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "telex")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "telex")
}

func New(root string) Store {
	if root == "" {
		root = DefaultRoot()
	}
	return Store{Root: root}
}

func (s Store) MailRoot() string { return filepath.Join(s.Root, "mail") }

func (s Store) MailboxPath(domainName, localPart string) (string, error) {
	if err := validatePathName("domain", domainName); err != nil {
		return "", err
	}
	if err := validatePathName("mailbox", localPart); err != nil {
		return "", err
	}
	return filepath.Join(s.MailRoot(), domainName, localPart), nil
}

func (s Store) SyncMailboxes(bootstrap *mail.MailboxBootstrap, syncedAt time.Time) (*SyncResult, error) {
	if bootstrap == nil {
		return nil, fmt.Errorf("mailbox bootstrap is required")
	}
	domains := domainsByID(bootstrap.Domains)
	result := &SyncResult{}
	for _, inbox := range bootstrap.Inboxes {
		if !inbox.Active {
			result.Skipped = append(result.Skipped, inbox)
			continue
		}
		domain, ok := domains[inbox.DomainID]
		if !ok {
			return nil, fmt.Errorf("domain %d not found for inbox %d", inbox.DomainID, inbox.ID)
		}
		meta := MailboxMeta{
			SchemaVersion: SchemaVersion,
			DomainID:      domain.ID,
			DomainName:    domain.Name,
			InboxID:       inbox.ID,
			Address:       inbox.Address,
			LocalPart:     inbox.LocalPart,
			Active:        inbox.Active,
			SyncedAt:      syncedAt,
		}
		if err := s.CreateMailbox(meta); err != nil {
			return nil, err
		}
		result.Created = append(result.Created, meta)
	}
	return result, nil
}

func (s Store) CreateMailbox(meta MailboxMeta) error {
	if meta.SchemaVersion == 0 {
		meta.SchemaVersion = SchemaVersion
	}
	path, err := s.MailboxPath(meta.DomainName, meta.LocalPart)
	if err != nil {
		return err
	}
	for _, dir := range append([]string{""}, mailboxDirs...) {
		if err := os.MkdirAll(filepath.Join(path, dir), 0o700); err != nil {
			return err
		}
	}
	return writeTOML(filepath.Join(path, "meta.toml"), meta)
}

func ReadMailboxMeta(path string) (*MailboxMeta, error) {
	var meta MailboxMeta
	if _, err := toml.DecodeFile(filepath.Join(path, "meta.toml"), &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (s Store) ListMailboxes() ([]MailboxMeta, error) {
	root := s.MailRoot()
	mailboxes := []MailboxMeta{}
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() || path == root {
			return nil
		}
		if _, err := os.Stat(filepath.Join(path, "meta.toml")); err != nil {
			return nil
		}
		meta, err := ReadMailboxMeta(path)
		if err != nil {
			return err
		}
		mailboxes = append(mailboxes, *meta)
		return filepath.SkipDir
	}); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	sort.Slice(mailboxes, func(i, j int) bool { return mailboxes[i].Address < mailboxes[j].Address })
	return mailboxes, nil
}

func domainsByID(domains []mail.Domain) map[int64]mail.Domain {
	indexed := make(map[int64]mail.Domain, len(domains))
	for _, domain := range domains {
		indexed[domain.ID] = domain
	}
	return indexed
}

func validatePathName(label, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%s name is required", label)
	}
	if value == "." || value == ".." || strings.ContainsRune(value, os.PathSeparator) {
		return fmt.Errorf("%s name %q is not safe for a path", label, value)
	}
	for _, r := range value {
		if r < 32 || r == 127 {
			return fmt.Errorf("%s name %q contains control characters", label, value)
		}
	}
	return nil
}

func writeTOML(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if err := toml.NewEncoder(f).Encode(value); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return nil
}
