package screens

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/mailstore"
)

func NewMail(store mailstore.Store) Mail {
	return NewMailWithActions(store, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

func NewAggregateMail(store mailstore.Store, title, box string, unreadOnly bool) Mail {
	m := NewMail(store)
	m.scope = MailScope{Title: title, Box: box, UnreadOnly: unreadOnly, Aggregate: true}
	return m
}

func NewAggregateMailWithActions(store mailstore.Store, title, box string, unreadOnly bool, toggleRead ToggleReadFunc, toggleStar ToggleStarFunc, archive MessageActionFunc, trash MessageActionFunc, restore MessageActionFunc, sync SyncFunc, sendDraft SendDraftFunc, updateDraft UpdateDraftFunc, deleteDraft DeleteDraftFunc, forward ForwardFunc, download DownloadAttachmentFunc, remoteSearch ...RemoteSearchFunc) Mail {
	m := NewMailWithActions(store, toggleRead, toggleStar, archive, trash, restore, sync, sendDraft, updateDraft, deleteDraft, forward, download, remoteSearch...)
	m.scope = MailScope{Title: title, Box: box, UnreadOnly: unreadOnly, Aggregate: true}
	return m
}

func (m Mail) WithStarredOnly() Mail {
	m.scope.StarredOnly = true
	return m
}

func NewMailWithActions(store mailstore.Store, toggleRead ToggleReadFunc, toggleStar ToggleStarFunc, archive MessageActionFunc, trash MessageActionFunc, restore MessageActionFunc, sync SyncFunc, sendDraft SendDraftFunc, updateDraft UpdateDraftFunc, deleteDraft DeleteDraftFunc, forward ForwardFunc, download DownloadAttachmentFunc, remoteSearch ...RemoteSearchFunc) Mail {
	var search RemoteSearchFunc
	if len(remoteSearch) > 0 {
		search = remoteSearch[0]
	}
	return Mail{store: store, toggleRead: toggleRead, toggleStar: toggleStar, archive: archive, trash: trash, restore: restore, sync: sync, sendDraft: sendDraft, updateDraft: updateDraft, deleteDraft: deleteDraft, forward: forward, download: download, remoteSearch: search, detailViewport: viewport.New(), articleViewport: viewport.New(), conversationViewport: viewport.New(), keys: DefaultMailKeyMap(), loading: true}
}

func (m Mail) WithConversationActions(conversation ConversationFunc, body ConversationBodyFunc) Mail {
	m.conversation = conversation
	m.conversationBody = body
	return m
}

func (m Mail) WithJunkActions(junk MessageActionFunc, notJunk MessageActionFunc) Mail {
	m.junk = junk
	m.notJunk = notJunk
	return m
}

func (m Mail) WithSenderPolicyActions(blockSender, unblockSender, blockDomain, unblockDomain, trustSender, untrustSender MessageActionFunc) Mail {
	m.blockSender = blockSender
	m.unblockSender = unblockSender
	m.blockDomain = blockDomain
	m.unblockDomain = unblockDomain
	m.trustSender = trustSender
	m.untrustSender = untrustSender
	return m
}

func (m Mail) Init() tea.Cmd { return m.loadCmd() }

func (m Mail) Title() string {
	if m.scope.Title != "" {
		return m.scope.Title
	}
	return "Mail"
}
