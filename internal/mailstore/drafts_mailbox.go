package mailstore

import (
	"fmt"
	"strings"
)

func (s Store) FindMailboxByAddress(address string) (*MailboxMeta, string, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, "", fmt.Errorf("mailbox address is required")
	}
	parts := strings.Split(address, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, "", fmt.Errorf("mailbox must be an address like hello@example.com")
	}
	path, err := s.MailboxPath(parts[1], parts[0])
	if err != nil {
		return nil, "", err
	}
	meta, err := ReadMailboxMeta(path)
	if err != nil {
		return nil, "", fmt.Errorf("mailbox %s has not been synced: %w", address, err)
	}
	if !strings.EqualFold(meta.Address, address) {
		return nil, "", fmt.Errorf("mailbox metadata address mismatch: found %s", meta.Address)
	}
	return meta, path, nil
}
