package screens

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"fmt"
	"github.com/elpdev/telex-cli/internal/mailstore"
	"os/exec"
)

func (m Mail) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if m.filePickerActive {
		return m.handleAttachFileMsg(msg)
	}

	switch msg := msg.(type) {
	case mailLoadedMsg:
		m.loading = false
		m.remoteResults = false
		m.err = msg.err
		m.status = ""
		if msg.err == nil {
			m.mailboxes = msg.mailboxes
			m.allMessages = msg.messages
			m.applySearch()
			m.clampSelection()
		}
		return m, nil
	case mailSyncedMsg:
		m.loading = false
		m.syncing = false
		m.remoteResults = false
		m.err = msg.loaded.err
		if msg.loaded.err == nil {
			m.mailboxes = msg.loaded.mailboxes
			m.allMessages = msg.loaded.messages
			m.applySearch()
			m.clampSelection()
		}
		if msg.err != nil {
			m.status = fmt.Sprintf("Sync failed: %v", msg.err)
			return m, nil
		}
		m.status = syncStatus(msg.result)
		return m, nil
	case remoteSearchLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.status = fmt.Sprintf("Remote search failed: %v", msg.err)
			return m, nil
		}
		m.remoteResults = true
		m.remoteSearchQuery = msg.query
		m.searchQuery = ""
		m.allMessages = msg.messages
		m.applySearch()
		m.messageIndex = 0
		m.clampSelection()
		m.status = fmt.Sprintf("Remote search: %s (%d result(s), transient)", msg.query, len(msg.messages))
		return m, nil
	case conversationLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not load conversation: %v", msg.err)
			return m, nil
		}
		m.conversationID = msg.conversationID
		m.conversationItems = msg.entries
		m.conversationIndex = 0
		m.resetConversationViewport()
		m.conversationBodyCache = make(map[string]string)
		m.mode = mailModeConversation
		m.status = ""
		m.clampConversationSelection()
		return m, m.loadConversationBodyCmd()
	case conversationBodyLoadedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not load conversation body: %v", msg.err)
			return m, nil
		}
		if m.conversationBodyCache == nil {
			m.conversationBodyCache = make(map[string]string)
		}
		m.conversationBodyCache[msg.key] = msg.body
		m.status = ""
		return m, nil
	case htmlOpenFinishedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not open HTML: %v", msg.err)
		} else {
			m.status = fmt.Sprintf("Opened HTML: %s", msg.path)
		}
		return m, nil
	case linkOpenFinishedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not open link: %v", msg.err)
		} else {
			m.status = fmt.Sprintf("Opened link: %s", msg.url)
		}
		return m, nil
	case linkCopyFinishedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not copy link: %v", msg.err)
		} else {
			m.status = "Copied link"
		}
		return m, nil
	case articleExtractedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not extract article: %v", msg.err)
			return m, nil
		}
		m.article = msg.article
		m.articleURL = msg.url
		m.resetArticleViewport()
		m.status = ""
		m.mode = mailModeArticle
		return m, nil
	case messageReadToggledMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not update read state: %v", msg.err)
			return m, nil
		}
		if m.scope.Aggregate && m.scope.UnreadOnly && msg.read {
			m.removeMessageByPath(msg.path)
			m.mode = mailModeList
			m.resetDetailViewport()
			m.clampSelection()
		} else {
			m.updateMessageByPath(msg.path, func(message *mailstore.CachedMessage) { message.Meta.Read = msg.read })
		}
		if msg.read {
			m.status = "Marked read"
		} else {
			m.status = "Marked unread"
		}
		return m, nil
	case messageStarToggledMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not update star state: %v", msg.err)
			return m, nil
		}
		m.updateMessageByPath(msg.path, func(message *mailstore.CachedMessage) { message.Meta.Starred = msg.starred })
		if msg.starred {
			m.status = "Starred"
		} else {
			m.status = "Unstarred"
		}
		return m, nil
	case messageMovedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not %s message: %v", msg.action, msg.err)
			return m, nil
		}
		m.removeMessageByPath(msg.path)
		m.mode = mailModeList
		m.resetDetailViewport()
		m.clampSelection()
		switch msg.action {
		case "archive":
			m.status = "Archived"
		case "junk":
			m.status = "Moved to junk"
		case "not-junk":
			m.status = "Moved to inbox"
		case "trash":
			m.status = "Moved to trash"
		case "restore":
			m.status = "Restored"
		}
		return m, nil
	case messagePolicyUpdatedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not update sender policy: %v", msg.err)
			return m, nil
		}
		m.updateMessageByPath(msg.path, func(message *mailstore.CachedMessage) {
			switch msg.action {
			case "block-sender":
				message.Meta.SenderBlocked = true
				message.Meta.SenderTrusted = false
			case "unblock-sender":
				message.Meta.SenderBlocked = false
			case "trust-sender":
				message.Meta.SenderTrusted = true
				message.Meta.SenderBlocked = false
			case "untrust-sender":
				message.Meta.SenderTrusted = false
			case "block-domain":
				message.Meta.DomainBlocked = true
			case "unblock-domain":
				message.Meta.DomainBlocked = false
			}
		})
		m.status = policyStatus(msg.action)
		return m, nil
	case draftEditedMsg:
		m.loading = false
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not save draft: %v", msg.err)
			return m, nil
		}
		if msg.path == "" && msg.existingPath != "" {
			draft, err := mailstore.ReadDraft(msg.existingPath)
			if err != nil {
				m.status = fmt.Sprintf("Could not save draft: %v", err)
				return m, nil
			}
			if m.currentBox() == "drafts" {
				loaded := m.load(m.mailboxIndex, m.currentBox())
				m.allMessages = loaded.messages
				m.applySearch()
				m.clampSelection()
			}
			m.status = fmt.Sprintf("Draft saved: %s", draft.Meta.ID)
			if draft.Meta.RemoteID > 0 && m.updateDraft != nil {
				m.status = fmt.Sprintf("Draft saved locally; syncing remote draft %d...", draft.Meta.RemoteID)
				return m, func() tea.Msg {
					return remoteDraftUpdatedMsg{remoteID: draft.Meta.RemoteID, err: m.updateDraft(context.Background(), *draft)}
				}
			}
			return m, nil
		}
		draft, err := saveEditedDraft(m.store, msg.mailbox, msg.path, msg.existingPath)
		if err != nil {
			m.status = fmt.Sprintf("Could not save draft: %v", err)
			return m, nil
		}
		m.status = fmt.Sprintf("Draft saved: %s", draft.Meta.ID)
		if msg.existingPath != "" && m.currentBox() == "drafts" {
			loaded := m.load(m.mailboxIndex, m.currentBox())
			m.allMessages = loaded.messages
			m.applySearch()
			m.clampSelection()
		}
		if draft.Meta.RemoteID > 0 && m.updateDraft != nil {
			m.status = fmt.Sprintf("Draft saved locally; syncing remote draft %d...", draft.Meta.RemoteID)
			return m, func() tea.Msg {
				return remoteDraftUpdatedMsg{remoteID: draft.Meta.RemoteID, err: m.updateDraft(context.Background(), *draft)}
			}
		}
		if draft.Meta.DraftKind == "forward" && draft.Meta.SourceMessageID > 0 && m.forward != nil {
			m.status = "Creating reviewed forward draft..."
			return m, func() tea.Msg {
				remoteID, status, err := m.forward(context.Background(), draft.Meta.SourceMessageID, *draft)
				return forwardDraftCreatedMsg{remoteID: remoteID, status: status, err: err}
			}
		}
		return m, nil
	case forwardDraftCreatedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not create forward draft: %v", msg.err)
			return m, nil
		}
		m.status = fmt.Sprintf("Forward draft created remotely: %d (%s)", msg.remoteID, msg.status)
		return m, nil
	case remoteDraftUpdatedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not sync remote draft %d: %v", msg.remoteID, msg.err)
			return m, nil
		}
		m.status = fmt.Sprintf("Remote draft synced: %d", msg.remoteID)
		return m, nil
	case draftSentMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not send draft: %v", msg.err)
			return m, nil
		}
		m.removeMessageByPath(msg.path)
		m.clampSelection()
		m.status = "Draft sent"
		return m, nil
	case draftDeletedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not delete draft: %v", msg.err)
			return m, nil
		}
		m.removeMessageByPath(msg.path)
		m.clampSelection()
		m.status = "Draft deleted"
		return m, nil
	case draftAttachmentDetachedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not detach attachment: %v", msg.err)
			return m, nil
		}
		loaded := m.load(m.mailboxIndex, m.currentBox())
		m.allMessages = loaded.messages
		m.applySearch()
		m.clampSelection()
		m.mode = mailModeDetail
		m.status = "Attachment detached"
		return m, nil
	case attachmentDownloadedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not save attachment: %v", msg.err)
			return m, nil
		}
		if msg.open {
			m.status = "Opening attachment..."
			cmd := exec.Command("xdg-open", msg.path)
			return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return attachmentOpenedMsg{path: msg.path, err: err} })
		}
		m.status = fmt.Sprintf("Saved attachment: %s", msg.path)
		return m, nil
	case attachmentOpenedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not open attachment: %v", msg.err)
		} else {
			m.status = fmt.Sprintf("Opened attachment: %s", msg.path)
		}
		return m, nil
	case MailActionMsg:
		return m.handleAction(msg.Action)
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}
