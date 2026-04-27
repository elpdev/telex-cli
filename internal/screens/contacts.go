package screens

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/contacts"
	"github.com/elpdev/telex-cli/internal/contactstore"
)

type ContactsSyncFunc func(context.Context) (ContactsSyncResult, error)
type DeleteContactFunc func(context.Context, int64) error
type LoadContactNoteFunc func(context.Context, int64) (*contacts.ContactNote, error)
type UpdateContactNoteFunc func(context.Context, int64, contacts.ContactNoteInput) (*contacts.ContactNote, error)
type LoadContactCommunicationsFunc func(context.Context, int64) ([]contacts.ContactCommunication, error)

type ContactsSyncResult struct {
	Contacts int
	Notes    int
}

type Contacts struct {
	store              contactstore.Store
	sync               ContactsSyncFunc
	delete             DeleteContactFunc
	loadNote           LoadContactNoteFunc
	updateNote         UpdateContactNoteFunc
	loadCommunications LoadContactCommunicationsFunc
	contacts           []contactstore.CachedContact
	contactList        list.Model
	detailViewport     viewport.Model
	index              int
	detail             *contactstore.CachedContact
	filter             string
	editing            bool
	confirm            string
	loading            bool
	syncing            bool
	err                error
	status             string
	keys               ContactsKeyMap
}

type ContactsKeyMap struct {
	Up             key.Binding
	Down           key.Binding
	Open           key.Binding
	Back           key.Binding
	Refresh        key.Binding
	Sync           key.Binding
	Search         key.Binding
	Delete         key.Binding
	EditNote       key.Binding
	Note           key.Binding
	Communications key.Binding
}

type contactsLoadedMsg struct {
	contacts []contactstore.CachedContact
	err      error
}

type contactsSyncedMsg struct {
	result ContactsSyncResult
	loaded contactsLoadedMsg
	err    error
}

type contactActionFinishedMsg struct {
	status string
	loaded contactsLoadedMsg
	err    error
}

type ContactsActionMsg struct{ Action string }

type ContactsSelection struct {
	Kind    string
	Subject string
	HasItem bool
}

type contactListItem struct {
	contact contactstore.CachedContact
}

func (i contactListItem) FilterValue() string { return i.contact.Meta.DisplayName }

func NewContacts(store contactstore.Store, sync ContactsSyncFunc) Contacts {
	return Contacts{store: store, sync: sync, loading: true, contactList: newContactsList(nil, 0, 0, 0), detailViewport: viewport.New(), keys: DefaultContactsKeyMap()}
}

func (c Contacts) WithActions(delete DeleteContactFunc, loadNote LoadContactNoteFunc, updateNote UpdateContactNoteFunc, loadCommunications LoadContactCommunicationsFunc) Contacts {
	c.delete = delete
	c.loadNote = loadNote
	c.updateNote = updateNote
	c.loadCommunications = loadCommunications
	return c
}

func DefaultContactsKeyMap() ContactsKeyMap {
	return ContactsKeyMap{
		Up:             key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "contact up")),
		Down:           key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "contact down")),
		Open:           key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		Back:           key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
		Refresh:        key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload cache")),
		Sync:           key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "sync contacts")),
		Search:         key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Delete:         key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete")),
		EditNote:       key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit note")),
		Note:           key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "refresh note")),
		Communications: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "communications")),
	}
}

func (c Contacts) Init() tea.Cmd { return c.loadCmd() }

func (c Contacts) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case contactsLoadedMsg:
		c.loading = false
		c.err = msg.err
		if msg.err == nil {
			c.contacts = msg.contacts
			c.clampIndex()
			c.refreshDetail()
		}
		return c, nil
	case contactsSyncedMsg:
		c.syncing = false
		c.err = msg.err
		if msg.err == nil {
			c.status = fmt.Sprintf("Synced %d contact(s), %d note(s)", msg.result.Contacts, msg.result.Notes)
			c.contacts = msg.loaded.contacts
			c.clampIndex()
			c.refreshDetail()
		} else {
			c.status = ""
		}
		return c, nil
	case contactActionFinishedMsg:
		c.loading = false
		c.err = msg.err
		if msg.err == nil {
			c.status = msg.status
			c.contacts = msg.loaded.contacts
			c.clampIndex()
			c.refreshDetail()
		} else {
			c.status = fmt.Sprintf("Contacts action failed: %v", msg.err)
		}
		return c, nil
	case ContactsActionMsg:
		return c.handleAction(msg.Action)
	case tea.KeyPressMsg:
		return c.handleKey(msg)
	}
	return c, nil
}

func (c Contacts) View(width, height int) string {
	style := lipgloss.NewStyle().Width(width).Height(height)
	if c.loading {
		return style.Render("Loading local contacts cache...")
	}
	if c.err != nil {
		return style.Render(fmt.Sprintf("Contacts cache error: %v\n\nRun `telex contacts sync` or press S to populate Contacts.", c.err))
	}
	var b strings.Builder
	b.WriteString("Contacts")
	b.WriteString(fmt.Sprintf(" · %d cached", len(c.contacts)))
	if c.filter != "" {
		b.WriteString(" · filter: " + c.filter)
	}
	b.WriteString("\n")
	if c.status != "" {
		b.WriteString(c.status + "\n")
	}
	if c.syncing {
		b.WriteString("Syncing remote Contacts...\n")
	}
	if c.editing {
		b.WriteString("Filter: " + c.filter + "\n")
	}
	if c.confirm != "" {
		b.WriteString(c.confirm + " [y/N]\n")
	}
	b.WriteString("\n")
	if c.detail != nil {
		headerLines := strings.Count(b.String(), "\n")
		b.WriteString(c.detailView(width, max(1, height-headerLines)))
		return style.Render(b.String())
	}
	visible := c.visibleContacts()
	if len(visible) == 0 {
		b.WriteString("No cached contacts found. Press S to sync.\n")
		return style.Render(b.String())
	}
	headerLines := strings.Count(b.String(), "\n")
	b.WriteString(c.renderContactsList(visible, width, max(1, height-headerLines)))
	return style.Render(b.String())
}

func (c Contacts) Title() string { return "Contacts" }

func (c Contacts) KeyBindings() []key.Binding {
	return []key.Binding{c.keys.Up, c.keys.Down, c.keys.Open, c.keys.Back, c.keys.Refresh, c.keys.Sync, c.keys.Search, c.keys.Delete, c.keys.EditNote, c.keys.Note, c.keys.Communications}
}

func (c Contacts) Selection() ContactsSelection {
	selected := c.selectedContact()
	if selected == nil {
		return ContactsSelection{Kind: "contact"}
	}
	return ContactsSelection{Kind: "contact", Subject: selected.Meta.DisplayName, HasItem: true}
}

func (c Contacts) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if c.confirm != "" {
		switch msg.String() {
		case "y", "Y":
			id := c.selectedID()
			c.confirm = ""
			if id == 0 || c.delete == nil {
				return c, nil
			}
			c.loading = true
			return c, c.deleteCmd(id)
		default:
			c.confirm = ""
			return c, nil
		}
	}
	if c.editing {
		switch msg.String() {
		case "enter":
			c.editing = false
			c.index = 0
			return c, nil
		case "esc":
			c.editing = false
			c.filter = ""
			c.index = 0
			return c, nil
		case "backspace":
			if len(c.filter) > 0 {
				c.filter = c.filter[:len(c.filter)-1]
				c.clampIndex()
			}
			return c, nil
		default:
			if len(msg.String()) == 1 {
				c.filter += msg.String()
				c.clampIndex()
			}
			return c, nil
		}
	}
	switch {
	case key.Matches(msg, c.keys.Up):
		if c.detail != nil {
			c.detailViewport.ScrollUp(1)
		} else if c.index > 0 {
			c.index--
			c.syncContactList()
		}
	case key.Matches(msg, c.keys.Down):
		if c.detail != nil {
			c.detailViewport.ScrollDown(1)
		} else if c.index < len(c.visibleContacts())-1 {
			c.index++
			c.syncContactList()
		}
	case key.Matches(msg, c.keys.Open):
		if c.detail == nil {
			c.detail = c.selectedContact()
			c.detailViewport = viewport.New()
		}
	case key.Matches(msg, c.keys.Back):
		c.detail = nil
		c.detailViewport = viewport.New()
	case key.Matches(msg, c.keys.Refresh):
		c.loading = true
		return c, c.loadCmd()
	case key.Matches(msg, c.keys.Sync):
		return c.handleAction("sync")
	case key.Matches(msg, c.keys.Search):
		c.editing = true
	case key.Matches(msg, c.keys.Delete):
		if selected := c.selectedContact(); selected != nil {
			c.confirm = fmt.Sprintf("Delete contact %s?", selected.Meta.DisplayName)
		}
	case key.Matches(msg, c.keys.EditNote):
		return c.handleAction("edit-note")
	case key.Matches(msg, c.keys.Note):
		return c.handleAction("refresh-note")
	case key.Matches(msg, c.keys.Communications):
		return c.handleAction("communications")
	}
	return c, nil
}

func (c Contacts) handleAction(action string) (Screen, tea.Cmd) {
	switch action {
	case "sync":
		if c.sync == nil {
			return c, nil
		}
		c.syncing = true
		return c, c.syncCmd()
	case "search":
		c.editing = true
		return c, nil
	case "delete":
		if selected := c.selectedContact(); selected != nil {
			c.confirm = fmt.Sprintf("Delete contact %s?", selected.Meta.DisplayName)
		}
		return c, nil
	case "edit-note":
		id := c.selectedID()
		if id == 0 || c.updateNote == nil {
			return c, nil
		}
		c.loading = true
		return c, c.editNoteCmd(id)
	case "refresh-note":
		id := c.selectedID()
		if id == 0 || c.loadNote == nil {
			return c, nil
		}
		c.loading = true
		return c, c.loadNoteCmd(id)
	case "communications":
		id := c.selectedID()
		if id == 0 || c.loadCommunications == nil {
			return c, nil
		}
		c.loading = true
		return c, c.communicationsCmd(id)
	}
	return c, nil
}

func (c Contacts) loadCmd() tea.Cmd {
	return func() tea.Msg {
		items, err := c.store.ListContacts()
		return contactsLoadedMsg{contacts: items, err: err}
	}
}

func (c Contacts) syncCmd() tea.Cmd {
	return func() tea.Msg {
		result, err := c.sync(context.Background())
		items, loadErr := c.store.ListContacts()
		if err == nil {
			err = loadErr
		}
		return contactsSyncedMsg{result: result, loaded: contactsLoadedMsg{contacts: items, err: loadErr}, err: err}
	}
}

func (c Contacts) deleteCmd(id int64) tea.Cmd {
	return func() tea.Msg {
		err := c.delete(context.Background(), id)
		items, loadErr := c.store.ListContacts()
		if err == nil {
			err = loadErr
		}
		return contactActionFinishedMsg{status: "Deleted contact " + strconv.FormatInt(id, 10), loaded: contactsLoadedMsg{contacts: items, err: loadErr}, err: err}
	}
}

func (c Contacts) loadNoteCmd(id int64) tea.Cmd {
	return func() tea.Msg {
		_, err := c.loadNote(context.Background(), id)
		items, loadErr := c.store.ListContacts()
		if err == nil {
			err = loadErr
		}
		return contactActionFinishedMsg{status: "Refreshed contact note", loaded: contactsLoadedMsg{contacts: items, err: loadErr}, err: err}
	}
}

func (c Contacts) editNoteCmd(id int64) tea.Cmd {
	selected := c.selectedContact()
	if c.detail != nil {
		selected = c.detail
	}
	title := defaultTitle
	body := ""
	if selected != nil {
		title = selected.Meta.DisplayName
		if selected.Note != nil {
			title = selected.Note.Meta.Title
			body = selected.Note.Body
		}
	}
	return func() tea.Msg {
		input, err := editContactNoteTemplate(title, body)
		if err != nil {
			return contactActionFinishedMsg{err: err}
		}
		_, err = c.updateNote(context.Background(), id, input)
		items, loadErr := c.store.ListContacts()
		if err == nil {
			err = loadErr
		}
		return contactActionFinishedMsg{status: "Updated contact note", loaded: contactsLoadedMsg{contacts: items, err: loadErr}, err: err}
	}
}

func (c Contacts) communicationsCmd(id int64) tea.Cmd {
	return func() tea.Msg {
		communications, err := c.loadCommunications(context.Background(), id)
		items, loadErr := c.store.ListContacts()
		if err == nil {
			err = loadErr
		}
		return contactActionFinishedMsg{status: fmt.Sprintf("Loaded %d communication(s)", len(communications)), loaded: contactsLoadedMsg{contacts: items, err: loadErr}, err: err}
	}
}

func (c Contacts) visibleContacts() []contactstore.CachedContact {
	if c.filter == "" {
		return c.contacts
	}
	query := strings.ToLower(c.filter)
	out := []contactstore.CachedContact{}
	for _, contact := range c.contacts {
		text := strings.ToLower(strings.Join([]string{contact.Meta.DisplayName, contact.Meta.PrimaryEmailAddress, contact.Meta.CompanyName, contact.Meta.Title, contact.Meta.Phone}, " "))
		if strings.Contains(text, query) {
			out = append(out, contact)
		}
	}
	return out
}

func (c *Contacts) clampIndex() {
	visible := c.visibleContacts()
	if c.index >= len(visible) {
		c.index = len(visible) - 1
	}
	if c.index < 0 {
		c.index = 0
	}
	c.syncContactList()
}

func (c *Contacts) refreshDetail() {
	if c.detail == nil {
		return
	}
	id := c.detail.Meta.RemoteID
	c.detail = nil
	for i := range c.contacts {
		if c.contacts[i].Meta.RemoteID == id {
			contact := c.contacts[i]
			c.detail = &contact
			return
		}
	}
}

func (c Contacts) selectedContact() *contactstore.CachedContact {
	visible := c.visibleContacts()
	if len(visible) == 0 || c.index < 0 || c.index >= len(visible) {
		return nil
	}
	selected := visible[c.index]
	return &selected
}

func (c Contacts) selectedID() int64 {
	if c.detail != nil {
		return c.detail.Meta.RemoteID
	}
	if selected := c.selectedContact(); selected != nil {
		return selected.Meta.RemoteID
	}
	return 0
}

func (c Contacts) renderContactsList(contacts []contactstore.CachedContact, width, height int) string {
	c.ensureContactList(contacts)
	c.contactList.SetSize(width, height)
	return c.contactList.View()
}

func (c *Contacts) syncContactList() {
	c.contactList = newContactsList(c.visibleContacts(), c.index, c.contactList.Width(), c.contactList.Height())
}

func (c Contacts) ensureContactList(contacts []contactstore.CachedContact) {
	if len(c.contactList.Items()) != len(contacts) {
		c.contactList = newContactsList(contacts, c.index, c.contactList.Width(), c.contactList.Height())
		return
	}
	c.contactList.Select(c.index)
}

func newContactsList(contacts []contactstore.CachedContact, selected, width, height int) list.Model {
	items := make([]list.Item, 0, len(contacts))
	for _, contact := range contacts {
		items = append(items, contactListItem{contact: contact})
	}
	return newSimpleList(items, contactListDelegate{}, selected, width, height)
}

type contactListDelegate struct{ simpleDelegate }

func (d contactListDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	contactItem, ok := item.(contactListItem)
	if !ok {
		return
	}
	_, _ = io.WriteString(w, formatContactRow(contactItem.contact, index == m.Index(), m.Width()))
}

func formatContactRow(contact contactstore.CachedContact, selected bool, width int) string {
	cursor := listCursor(selected)
	name := contact.Meta.DisplayName
	if name == "" {
		name = "Unnamed contact"
	}
	parts := []string{name}
	if contact.Meta.PrimaryEmailAddress != "" {
		parts = append(parts, "<"+contact.Meta.PrimaryEmailAddress+">")
	}
	if contact.Meta.CompanyName != "" {
		parts = append(parts, contact.Meta.CompanyName)
	}
	return cursor + truncate(strings.Join(parts, " · "), max(0, width-2))
}

func (c Contacts) detailView(width, height int) string {
	contact := c.detail
	var b strings.Builder
	b.WriteString(contact.Meta.DisplayName + "\n")
	b.WriteString("Type: " + contact.Meta.ContactType + "\n")
	if contact.Meta.PrimaryEmailAddress != "" {
		b.WriteString("Email: " + contact.Meta.PrimaryEmailAddress + "\n")
	}
	if contact.Meta.CompanyName != "" {
		b.WriteString("Company: " + contact.Meta.CompanyName + "\n")
	}
	if contact.Meta.Title != "" {
		b.WriteString("Title: " + contact.Meta.Title + "\n")
	}
	if contact.Meta.Phone != "" {
		b.WriteString("Phone: " + contact.Meta.Phone + "\n")
	}
	if contact.Meta.Website != "" {
		b.WriteString("Website: " + contact.Meta.Website + "\n")
	}
	if len(contact.Meta.EmailAddresses) > 1 {
		b.WriteString("\nEmail addresses\n")
		for _, email := range contact.Meta.EmailAddresses {
			label := email.Label
			if label == "" {
				label = "email"
			}
			b.WriteString("  " + label + ": " + email.EmailAddress + "\n")
		}
	}
	if contact.Note != nil && strings.TrimSpace(contact.Note.Body) != "" {
		b.WriteString("\nNote: " + contact.Note.Meta.Title + "\n")
		b.WriteString(contact.Note.Body + "\n")
	}
	if len(contact.Communications) > 0 {
		items := append([]contactstore.CommunicationMeta(nil), contact.Communications...)
		sort.Slice(items, func(i, j int) bool { return items[i].OccurredAt.After(items[j].OccurredAt) })
		b.WriteString("\nCommunications\n")
		for _, item := range items {
			summary := item.Subject
			if summary == "" {
				summary = item.PreviewText
			}
			b.WriteString(fmt.Sprintf("  %s · %s · %s\n", item.OccurredAt.Format("2006-01-02"), item.Direction, summary))
		}
	} else {
		b.WriteString("\nPress e to edit note, c to load communications, or N to refresh note.\n")
	}
	body := b.String()
	c.detailViewport.SetWidth(width)
	c.detailViewport.SetHeight(height)
	c.detailViewport.SetContent(strings.TrimRight(body, "\n"))
	return c.detailViewport.View()
}

func editContactNoteTemplate(title, body string) (contacts.ContactNoteInput, error) {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("telex-contact-note-%d.md", time.Now().UnixNano()))
	defer os.Remove(path)
	if err := os.WriteFile(path, []byte(fmt.Sprintf("Title: %s\n\n%s", title, body)), 0o600); err != nil {
		return contacts.ContactNoteInput{}, err
	}
	editor := contactsEditorCommand()
	if editor == "" {
		return contacts.ContactNoteInput{}, fmt.Errorf("set TELEX_CONTACTS_EDITOR, TELEX_NOTES_EDITOR, VISUAL, or EDITOR to edit contact notes")
	}
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return contacts.ContactNoteInput{}, fmt.Errorf("set TELEX_CONTACTS_EDITOR, TELEX_NOTES_EDITOR, VISUAL, or EDITOR to edit contact notes")
	}
	cmd := exec.Command(parts[0], append(parts[1:], path)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return contacts.ContactNoteInput{}, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return contacts.ContactNoteInput{}, err
	}
	return parseContactNoteTemplate(string(content)), nil
}

func contactsEditorCommand() string {
	if editor := strings.TrimSpace(os.Getenv("TELEX_CONTACTS_EDITOR")); editor != "" {
		return editor
	}
	if editor := strings.TrimSpace(os.Getenv("TELEX_NOTES_EDITOR")); editor != "" {
		return editor
	}
	if editor := strings.TrimSpace(os.Getenv("VISUAL")); editor != "" {
		return editor
	}
	return strings.TrimSpace(os.Getenv("EDITOR"))
}

func parseContactNoteTemplate(content string) contacts.ContactNoteInput {
	lines := strings.Split(content, "\n")
	title := defaultTitle
	start := 0
	if len(lines) > 0 && strings.HasPrefix(strings.ToLower(lines[0]), "title:") {
		if parsed := strings.TrimSpace(lines[0][len("Title:"):]); parsed != "" {
			title = parsed
		}
		start = 1
		if len(lines) > 1 && strings.TrimSpace(lines[1]) == "" {
			start = 2
		}
	}
	return contacts.ContactNoteInput{Title: title, Body: strings.Join(lines[start:], "\n")}
}
