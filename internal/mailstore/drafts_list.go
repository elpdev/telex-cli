package mailstore

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/BurntSushi/toml"
)

func DeleteDraft(path string) error {
	draft, err := ReadDraft(path)
	if err != nil {
		return err
	}
	if draft.Meta.Kind != "draft" {
		return fmt.Errorf("can only delete drafts, got %s", draft.Meta.Kind)
	}
	return os.RemoveAll(path)
}

func ListDrafts(mailboxPath string) ([]Draft, error) {
	entries, err := os.ReadDir(filepath.Join(mailboxPath, "drafts"))
	if err != nil {
		return nil, err
	}
	drafts := make([]Draft, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(mailboxPath, "drafts", entry.Name())
		draft, err := ReadDraft(path)
		if err != nil {
			return nil, err
		}
		drafts = append(drafts, *draft)
	}
	sort.Slice(drafts, func(i, j int) bool { return drafts[i].Meta.CreatedAt.After(drafts[j].Meta.CreatedAt) })
	return drafts, nil
}

func ListOutbox(mailboxPath string) ([]Draft, error) {
	return listItems(mailboxPath, "outbox")
}

func ListSent(mailboxPath string) ([]Draft, error) {
	return listItems(mailboxPath, "sent")
}

func listItems(mailboxPath, box string) ([]Draft, error) {
	root := filepath.Join(mailboxPath, box)
	drafts := []Draft{}
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
		draft, err := ReadDraft(path)
		if err != nil {
			return err
		}
		drafts = append(drafts, *draft)
		return filepath.SkipDir
	}); err != nil {
		return nil, err
	}
	sort.Slice(drafts, func(i, j int) bool { return drafts[i].Meta.UpdatedAt.After(drafts[j].Meta.UpdatedAt) })
	return drafts, nil
}

func ReadDraft(path string) (*Draft, error) {
	var meta DraftMeta
	if _, err := toml.DecodeFile(filepath.Join(path, "meta.toml"), &meta); err != nil {
		return nil, err
	}
	body, err := os.ReadFile(filepath.Join(path, "body.md"))
	if err != nil {
		return nil, err
	}
	return &Draft{Meta: meta, Path: path, Body: string(body)}, nil
}
