package screens

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/components/filepicker"
	"github.com/elpdev/telex-cli/internal/mailstore"
	"github.com/elpdev/telex-cli/internal/opener"
)

func (m Mail) handleAttachmentsKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if key.Matches(msg, m.keys.Back) {
		m.mode = mailModeDetail
		return m, nil
	}
	attachments := m.messages[m.messageIndex].Meta.Attachments
	if len(attachments) == 0 {
		return m, nil
	}
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.attachmentIndex > 0 {
			m.attachmentIndex--
		}
	case key.Matches(msg, m.keys.Down):
		if m.attachmentIndex < len(attachments)-1 {
			m.attachmentIndex++
		}
	case key.Matches(msg, m.keys.Open):
		return m.openAttachment()
	case key.Matches(msg, m.keys.Delete):
		return m.requestConfirm("detach-attachment", "Detach this attachment from the draft?")
	case key.Matches(msg, m.keys.Copy):
		return m.copyAttachmentURL()
	case key.Matches(msg, m.keys.Send):
		m.savingAttachment = true
		m.saveDirInput = defaultDownloadDir()
		m.status = "Save to: " + m.saveDirInput
	}
	return m, nil
}

func (m Mail) detachSelectedDraftAttachment() (Screen, tea.Cmd) {
	if m.currentBox() != "drafts" {
		m.status = "detach is only available from drafts"
		return m, nil
	}
	message := m.messages[m.messageIndex]
	attachment := message.Meta.Attachments[m.attachmentIndex]
	m.status = "Detaching attachment..."
	return m, func() tea.Msg {
		_, err := mailstore.DetachFileFromDraft(message.Path, attachmentFileLabel(attachment), time.Now())
		return draftAttachmentDetachedMsg{path: message.Path, err: err}
	}
}

func (m Mail) handleAttachmentSaveKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.savingAttachment = false
		m.saveDirInput = ""
		m.status = "Cancelled"
		return m, nil
	case "enter":
		dir := strings.TrimSpace(m.saveDirInput)
		m.savingAttachment = false
		m.saveDirInput = ""
		return m.saveAttachmentTo(dir)
	case "backspace":
		if len(m.saveDirInput) > 0 {
			m.saveDirInput = m.saveDirInput[:len(m.saveDirInput)-1]
		}
		m.status = "Save to: " + m.saveDirInput
		return m, nil
	}
	if msg.Text != "" {
		m.saveDirInput += msg.Text
		m.status = "Save to: " + m.saveDirInput
	}
	return m, nil
}

func (m Mail) handleAttachFileMsg(msg tea.Msg) (Screen, tea.Cmd) {
	picker, action, cmd := m.filePicker.Update(msg)
	m.filePicker = picker
	switch action.Type {
	case filepicker.ActionCancel:
		m.filePickerActive = false
		m.status = "Cancelled"
		return m, nil
	case filepicker.ActionSelect:
		m.filePickerActive = false
		return m.attachFileToSelectedDraft(action.Path)
	}
	if m.filePicker.Err != nil {
		m.status = fmt.Sprintf("File picker: %v", m.filePicker.Err)
	} else {
		m.status = "Select file to attach"
	}
	return m, cmd
}

func (m Mail) startAttachFile() (Screen, tea.Cmd) {
	if len(m.messages) == 0 || m.currentBox() != "drafts" {
		m.status = "attach is only available from drafts"
		return m, nil
	}
	cwd, err := os.Getwd()
	if err != nil || cwd == "" {
		cwd, _ = os.UserHomeDir()
	}
	m.filePicker = filepicker.New("", cwd, filepicker.ModeOpenFile)
	m.filePickerActive = true
	m.status = "Select file to attach"
	return m, m.filePicker.Init()
}

func (m Mail) attachFileToSelectedDraft(path string) (Screen, tea.Cmd) {
	if path == "" {
		m.status = "No file selected"
		return m, nil
	}
	if len(m.messages) == 0 || m.currentBox() != "drafts" {
		return m, nil
	}
	draftPath := m.messages[m.messageIndex].Path
	m.status = "Attaching file..."
	return m, func() tea.Msg {
		_, err := mailstore.AttachFileToDraft(draftPath, expandHome(path), time.Now())
		return draftEditedMsg{existingPath: draftPath, err: err}
	}
}

func (m Mail) openAttachment() (Screen, tea.Cmd) {
	attachment := m.messages[m.messageIndex].Meta.Attachments[m.attachmentIndex]
	path := mailstore.AttachmentCachePath(m.messages[m.messageIndex].Path, attachment)
	if _, err := os.Stat(path); err == nil {
		m.status = "Opening attachment..."
		cmd, err := opener.Command(path)
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return attachmentOpenedMsg{path: path, err: err} })
	}
	return m.downloadAttachment(path, true)
}

func (m Mail) saveAttachmentTo(dir string) (Screen, tea.Cmd) {
	if dir == "" {
		dir = defaultDownloadDir()
	}
	attachment := m.messages[m.messageIndex].Meta.Attachments[m.attachmentIndex]
	dest := uniquePath(filepath.Join(expandHome(dir), attachmentSaveName(attachment)))
	cachePath := mailstore.AttachmentCachePath(m.messages[m.messageIndex].Path, attachment)
	if data, err := os.ReadFile(cachePath); err == nil {
		m.status = "Saving attachment..."
		return m, func() tea.Msg { return attachmentDownloadedMsg{path: dest, err: writeAttachmentFile(dest, data)} }
	}
	return m.downloadAttachment(dest, false)
}

func (m Mail) downloadAttachment(path string, open bool) (Screen, tea.Cmd) {
	if m.download == nil {
		m.status = "Attachment download is not configured"
		return m, nil
	}
	attachment := m.messages[m.messageIndex].Meta.Attachments[m.attachmentIndex]
	m.status = "Downloading attachment..."
	return m, func() tea.Msg {
		data, err := m.download(context.Background(), attachment)
		if err != nil {
			return attachmentDownloadedMsg{path: path, open: open, err: err}
		}
		return attachmentDownloadedMsg{path: path, open: open, err: writeAttachmentFile(path, data)}
	}
}

func (m Mail) copyAttachmentURL() (Screen, tea.Cmd) {
	attachment := m.messages[m.messageIndex].Meta.Attachments[m.attachmentIndex]
	if attachment.DownloadURL == "" {
		m.status = "No download URL for this attachment"
		return m, nil
	}
	cmd, err := clipboardCommand(attachment.DownloadURL)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.status = "Copying attachment URL..."
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return linkCopyFinishedMsg{url: attachment.DownloadURL, err: err} })
}

func writeAttachmentFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func attachmentSaveName(attachment mailstore.AttachmentMeta) string {
	path := mailstore.AttachmentCachePath("", attachment)
	return filepath.Base(path)
}

func attachmentFileLabel(attachment mailstore.AttachmentMeta) string {
	if attachment.CacheName != "" {
		return attachment.CacheName
	}
	return attachment.Filename
}

func uniquePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func defaultDownloadDir() string {
	if xdg := strings.TrimSpace(os.Getenv("XDG_DOWNLOAD_DIR")); xdg != "" {
		return expandHome(xdg)
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, "Downloads")
	}
	return "."
}

func expandHome(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}
