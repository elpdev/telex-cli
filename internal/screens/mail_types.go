package screens

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	"github.com/elpdev/telex-cli/internal/articletext"
	"github.com/elpdev/telex-cli/internal/components/filepicker"
	"github.com/elpdev/telex-cli/internal/emailtext"
	"github.com/elpdev/telex-cli/internal/mailstore"
)

type mailMode int

const (
	mailModeList mailMode = iota
	mailModeDetail
	mailModeLinks
	mailModeArticle
	mailModeAttachments
	mailModeConversation
	mailModeComposeFrom
	mailReadWidth = 100
)

var mailBoxes = []string{"inbox", "archive", "trash", "sent", "outbox", "drafts", "junk"}

var extractArticleURL = articletext.NewExtractor().ExtractURL

type Mail struct {
	store                 mailstore.Store
	scope                 MailScope
	toggleRead            ToggleReadFunc
	toggleStar            ToggleStarFunc
	archive               MessageActionFunc
	trash                 MessageActionFunc
	junk                  MessageActionFunc
	notJunk               MessageActionFunc
	restore               MessageActionFunc
	blockSender           MessageActionFunc
	unblockSender         MessageActionFunc
	blockDomain           MessageActionFunc
	unblockDomain         MessageActionFunc
	trustSender           MessageActionFunc
	untrustSender         MessageActionFunc
	sync                  SyncFunc
	sendDraft             SendDraftFunc
	updateDraft           UpdateDraftFunc
	deleteDraft           DeleteDraftFunc
	forward               ForwardFunc
	download              DownloadAttachmentFunc
	remoteSearch          RemoteSearchFunc
	conversation          ConversationFunc
	conversationBody      ConversationBodyFunc
	mailboxes             []mailstore.MailboxMeta
	mailboxIndex          int
	boxIndex              int
	allMessages           []mailstore.CachedMessage
	messages              []mailstore.CachedMessage
	messageIndex          int
	searching             bool
	searchQuery           string
	searchInput           string
	remoteSearching       bool
	remoteSearchQuery     string
	remoteSearchInput     string
	remoteResults         bool
	detailViewport        viewport.Model
	links                 []emailtext.Link
	linkIndex             int
	attachmentIndex       int
	savingAttachment      bool
	saveDirInput          string
	filePickerActive      bool
	filePicker            filepicker.Model
	forwarding            bool
	forwardToInput        string
	article               string
	articleURL            string
	articleViewport       viewport.Model
	conversationID        int64
	conversationItems     []ConversationEntry
	conversationIndex     int
	conversationBodyCache map[string]string
	conversationViewport  viewport.Model
	composeFromIndex      int
	previousMode          mailMode
	mode                  mailMode
	loading               bool
	syncing               bool
	confirm               string
	err                   error
	status                string
	keys                  MailKeyMap
}

type MailScope struct {
	Title       string
	Box         string
	UnreadOnly  bool
	StarredOnly bool
	Aggregate   bool
}

type MailKeyMap struct {
	Up           key.Binding
	Down         key.Binding
	Previous     key.Binding
	Next         key.Binding
	BoxPrev      key.Binding
	BoxNext      key.Binding
	Open         key.Binding
	OpenHTML     key.Binding
	Links        key.Binding
	Extract      key.Binding
	Compose      key.Binding
	Reply        key.Binding
	Forward      key.Binding
	Send         key.Binding
	Delete       key.Binding
	Attachments  key.Binding
	ToggleRead   key.Binding
	ToggleStar   key.Binding
	Archive      key.Binding
	Junk         key.Binding
	NotJunk      key.Binding
	Trash        key.Binding
	Restore      key.Binding
	Copy         key.Binding
	Back         key.Binding
	Refresh      key.Binding
	RemoteSearch key.Binding
	Thread       key.Binding
}

type mailLoadedMsg struct {
	mailboxes []mailstore.MailboxMeta
	messages  []mailstore.CachedMessage
	err       error
}

type mailSyncedMsg struct {
	result MailSyncResult
	loaded mailLoadedMsg
	err    error
}

type remoteSearchLoadedMsg struct {
	query    string
	messages []mailstore.CachedMessage
	err      error
}

type conversationLoadedMsg struct {
	conversationID int64
	entries        []ConversationEntry
	err            error
}

type conversationBodyLoadedMsg struct {
	key  string
	body string
	err  error
}

type htmlOpenFinishedMsg struct {
	path string
	err  error
}

type linkOpenFinishedMsg struct {
	url string
	err error
}

type linkCopyFinishedMsg struct {
	url string
	err error
}

type articleExtractedMsg struct {
	url     string
	article string
	err     error
}

type messageReadToggledMsg struct {
	index int
	path  string
	read  bool
	err   error
}

type messageStarToggledMsg struct {
	index   int
	path    string
	starred bool
	err     error
}

type messageMovedMsg struct {
	index  int
	path   string
	action string
	err    error
}

type messagePolicyUpdatedMsg struct {
	path   string
	action string
	err    error
}

type draftEditedMsg struct {
	path         string
	existingPath string
	mailbox      mailstore.MailboxMeta
	err          error
}

type forwardDraftCreatedMsg struct {
	remoteID int64
	status   string
	err      error
}

type remoteDraftUpdatedMsg struct {
	remoteID int64
	err      error
}

type draftDeletedMsg struct {
	index int
	path  string
	err   error
}

type draftSentMsg struct {
	index int
	path  string
	err   error
}

type draftAttachmentDetachedMsg struct {
	path string
	err  error
}

type attachmentDownloadedMsg struct {
	path string
	open bool
	err  error
}

type attachmentOpenedMsg struct {
	path string
	err  error
}
