package screens

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/emailtext"
)

func (m Mail) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if m.confirm != "" {
		return m.handleConfirmKey(msg)
	}
	if m.searching {
		return m.handleSearchKey(msg)
	}
	if m.remoteSearching {
		return m.handleRemoteSearchKey(msg)
	}
	if m.savingAttachment {
		return m.handleAttachmentSaveKey(msg)
	}
	if m.filePickerActive {
		return m.handleAttachFileMsg(msg)
	}
	if m.forwarding {
		return m.handleForwardKey(msg)
	}
	if m.mode == mailModeComposeFrom {
		return m.handleComposeFromKey(msg)
	}
	if m.mode == mailModeArticle {
		return m.handleArticleKey(msg)
	}
	if m.mode == mailModeAttachments {
		return m.handleAttachmentsKey(msg)
	}
	if m.mode == mailModeConversation {
		return m.handleConversationKey(msg)
	}
	if m.mode == mailModeLinks {
		return m.handleLinksKey(msg)
	}
	if m.mode == mailModeDetail {
		if key.Matches(msg, m.keys.Back) {
			m.mode = mailModeList
			m.resetDetailViewport()
			m.status = ""
			return m.markSelectedRead()
		}
		if key.Matches(msg, m.keys.OpenHTML) {
			return m.openHTML()
		}
		if key.Matches(msg, m.keys.Links) {
			m.links = emailtext.Links(m.messages[m.messageIndex].BodyText, m.messages[m.messageIndex].BodyHTML)
			m.linkIndex = 0
			m.mode = mailModeLinks
			if len(m.links) == 0 {
				m.status = "No links found in this message"
			}
			return m, nil
		}
		if key.Matches(msg, m.keys.Attachments) {
			if len(m.messages[m.messageIndex].Meta.Attachments) == 0 {
				m.status = "No attachments on this message"
				return m, nil
			}
			m.attachmentIndex = 0
			m.mode = mailModeAttachments
			m.status = ""
			return m, nil
		}
		if key.Matches(msg, m.keys.Thread) {
			return m.openConversation()
		}
		if key.Matches(msg, m.keys.Reply) {
			return m.editReplyDraft()
		}
		if key.Matches(msg, m.keys.Forward) {
			return m.startForward()
		}
		if key.Matches(msg, m.keys.Send) {
			return m.requestConfirm("send-draft", "Send this draft?")
		}
		if key.Matches(msg, m.keys.Extract) {
			return m.editSelectedDraft()
		}
		if key.Matches(msg, m.keys.Delete) {
			return m.requestConfirm("delete-draft", "Delete this draft?")
		}
		if key.Matches(msg, m.keys.ToggleRead) {
			return m.toggleSelectedRead()
		}
		if key.Matches(msg, m.keys.ToggleStar) {
			return m.toggleSelectedStar()
		}
		if key.Matches(msg, m.keys.Archive) {
			return m.moveSelectedMessage("archive")
		}
		if key.Matches(msg, m.keys.Junk) {
			return m.moveSelectedMessage("junk")
		}
		if key.Matches(msg, m.keys.NotJunk) {
			return m.moveSelectedMessage("not-junk")
		}
		if key.Matches(msg, m.keys.Trash) {
			return m.requestConfirm("trash", "Move this message to trash?")
		}
		if key.Matches(msg, m.keys.Restore) {
			return m.moveSelectedMessage("restore")
		}
		switch {
		case key.Matches(msg, m.keys.Up):
			m.syncDetailViewport(mailReadWidth, 1)
			m.detailViewport.ScrollUp(1)
		case key.Matches(msg, m.keys.Down):
			m.syncDetailViewport(mailReadWidth, 1)
			m.detailViewport.ScrollDown(1)
		}
		return m, nil
	}
	if len(m.mailboxes) == 0 {
		return m, nil
	}
	switch {
	case key.Matches(msg, m.keys.Refresh):
		if m.sync == nil {
			m.loading = true
			return m, m.loadCmd()
		}
		if m.syncing {
			return m, nil
		}
		m.syncing = true
		m.status = "Syncing mailboxes, outbox, and inbox..."
		return m, m.syncCmd()
	case key.Matches(msg, m.keys.Compose):
		return m.editComposeDraft()
	case key.Matches(msg, m.keys.Send):
		return m.requestConfirm("send-draft", "Send this draft?")
	case key.Matches(msg, m.keys.Extract):
		return m.editSelectedDraft()
	case key.Matches(msg, m.keys.Delete):
		return m.requestConfirm("delete-draft", "Delete this draft?")
	case msg.String() == "/":
		m.searching = true
		m.searchInput = m.searchQuery
		m.status = "Search: " + m.searchInput
		return m, nil
	case key.Matches(msg, m.keys.RemoteSearch):
		if m.remoteSearch == nil {
			m.status = "Remote search is not configured"
			return m, nil
		}
		if m.scope.Aggregate {
			m.status = "Remote search is only available from a single mailbox"
			return m, nil
		}
		if !m.currentBoxSupportsRemoteSearch() {
			m.status = "Remote search is available for inbox, archive, and trash"
			return m, nil
		}
		m.remoteSearching = true
		m.remoteSearchInput = m.remoteSearchQuery
		m.status = "Remote search: " + m.remoteSearchInput
		return m, nil
	case key.Matches(msg, m.keys.BoxPrev):
		if m.scope.Aggregate {
			return m, nil
		}
		if m.boxIndex > 0 {
			m.boxIndex--
			m.messageIndex = 0
			m.loading = true
			return m, m.loadCmd()
		}
	case key.Matches(msg, m.keys.BoxNext):
		if m.scope.Aggregate {
			return m, nil
		}
		if m.boxIndex < len(mailBoxes)-1 {
			m.boxIndex++
			m.messageIndex = 0
			m.loading = true
			return m, m.loadCmd()
		}
	case key.Matches(msg, m.keys.Previous):
		if m.scope.Aggregate {
			return m, nil
		}
		if m.mailboxIndex > 0 {
			m.mailboxIndex--
			m.messageIndex = 0
			m.loading = true
			return m, m.loadCmd()
		}
	case key.Matches(msg, m.keys.Next):
		if m.scope.Aggregate {
			return m, nil
		}
		if m.mailboxIndex < len(m.mailboxes)-1 {
			m.mailboxIndex++
			m.messageIndex = 0
			m.loading = true
			return m, m.loadCmd()
		}
	case key.Matches(msg, m.keys.Up):
		if m.messageIndex > 0 {
			m.messageIndex--
		}
	case key.Matches(msg, m.keys.Down):
		if m.messageIndex < len(m.messages)-1 {
			m.messageIndex++
		}
	case key.Matches(msg, m.keys.Open):
		if len(m.messages) > 0 {
			m.mode = mailModeDetail
			m.resetDetailViewport()
			m.status = ""
		}
	case key.Matches(msg, m.keys.Thread):
		return m.openConversation()
	case key.Matches(msg, m.keys.ToggleRead):
		return m.toggleSelectedRead()
	case key.Matches(msg, m.keys.ToggleStar):
		return m.toggleSelectedStar()
	case key.Matches(msg, m.keys.Archive):
		if m.currentBox() == "drafts" {
			return m.startAttachFile()
		}
		return m.moveSelectedMessage("archive")
	case key.Matches(msg, m.keys.Junk):
		return m.moveSelectedMessage("junk")
	case key.Matches(msg, m.keys.NotJunk):
		return m.moveSelectedMessage("not-junk")
	case key.Matches(msg, m.keys.Trash):
		return m.requestConfirm("trash", "Move this message to trash?")
	case key.Matches(msg, m.keys.Restore):
		return m.moveSelectedMessage("restore")
	case key.Matches(msg, m.keys.Back):
		return m, nil
	}
	return m, nil
}

func (m Mail) handleConfirmKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "y":
		action := m.confirm
		m.confirm = ""
		switch action {
		case "trash":
			return m.moveSelectedMessage("trash")
		case "send-draft":
			return m.sendSelectedDraft()
		case "delete-draft":
			return m.deleteSelectedDraft()
		case "detach-attachment":
			return m.detachSelectedDraftAttachment()
		}
	case "n", "esc":
		m.confirm = ""
		m.status = "Cancelled"
		return m, nil
	}
	return m, nil
}

func (m Mail) requestConfirm(action, prompt string) (Screen, tea.Cmd) {
	m.confirm = action
	m.status = prompt + " Press y to confirm, n to cancel."
	return m, nil
}
