package mailstore

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func AttachFileToDraft(draftPath, sourcePath string, now time.Time) (*Draft, error) {
	draft, err := ReadDraft(draftPath)
	if err != nil {
		return nil, err
	}
	if draft.Meta.Kind != "draft" {
		return nil, fmt.Errorf("can only attach files to drafts, got %s", draft.Meta.Kind)
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("cannot attach directory %s", sourcePath)
	}
	if now.IsZero() {
		now = time.Now()
	}
	filename := filepath.Base(sourcePath)
	cacheName := uniqueAttachmentName(filepath.Join(draftPath, "attachments"), AttachmentMeta{Filename: filename})
	if err := copyFile(sourcePath, filepath.Join(draftPath, "attachments", cacheName)); err != nil {
		return nil, err
	}
	draft.Meta.Attachments = append(draft.Meta.Attachments, AttachmentMeta{Filename: filename, CacheName: cacheName, ByteSize: info.Size(), ContentType: "application/octet-stream"})
	draft.Meta.UpdatedAt = now
	if err := writeTOML(filepath.Join(draftPath, "meta.toml"), draft.Meta); err != nil {
		return nil, err
	}
	return ReadDraft(draftPath)
}

func DetachFileFromDraft(draftPath, name string, now time.Time) (*Draft, error) {
	draft, err := ReadDraft(draftPath)
	if err != nil {
		return nil, err
	}
	if draft.Meta.Kind != "draft" {
		return nil, fmt.Errorf("can only detach files from drafts, got %s", draft.Meta.Kind)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("attachment name is required")
	}
	index := -1
	for i, attachment := range draft.Meta.Attachments {
		if attachment.CacheName == name || attachment.Filename == name || attachmentFileName(attachment) == name {
			index = i
			break
		}
	}
	if index < 0 {
		return nil, fmt.Errorf("attachment %q was not found", name)
	}
	attachment := draft.Meta.Attachments[index]
	draft.Meta.Attachments = append(draft.Meta.Attachments[:index], draft.Meta.Attachments[index+1:]...)
	if now.IsZero() {
		now = time.Now()
	}
	draft.Meta.UpdatedAt = now
	if err := os.Remove(AttachmentCachePath(draftPath, attachment)); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := writeTOML(filepath.Join(draftPath, "meta.toml"), draft.Meta); err != nil {
		return nil, err
	}
	return ReadDraft(draftPath)
}

func uniqueAttachmentName(dir string, attachment AttachmentMeta) string {
	name := attachmentCacheName(attachment)
	if name == "" {
		name = "attachment"
	}
	if _, err := os.Stat(filepath.Join(dir, name)); os.IsNotExist(err) {
		return name
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, err := os.Stat(filepath.Join(dir, candidate)); os.IsNotExist(err) {
			return candidate
		}
	}
}

func copyFile(sourcePath, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o700); err != nil {
		return err
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()
	dest, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = dest.Close() }()
	_, err = io.Copy(dest, source)
	return err
}
