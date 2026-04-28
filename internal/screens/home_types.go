package screens

import (
	"context"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/hackernews/pkg/hn"
	"github.com/elpdev/telex-cli/internal/calendarstore"
	"github.com/elpdev/telex-cli/internal/components/card"
	"github.com/elpdev/telex-cli/internal/contactstore"
	"github.com/elpdev/telex-cli/internal/drivestore"
	"github.com/elpdev/telex-cli/internal/mailstore"
	"github.com/elpdev/telex-cli/internal/notestore"
	"github.com/elpdev/telex-cli/internal/taskstore"
	"github.com/elpdev/telex-cli/internal/theme"
)

type HomeNavigateFunc func(screenID string) tea.Cmd

// NewsFetcher returns up to limit top stories. Implementations typically wrap
// hn.Client.TopStories. May be nil — the news card then renders an empty state.
type NewsFetcher func(ctx context.Context, limit int) ([]hn.Item, error)

const (
	homeGridCols     = 2
	homeStackedBelow = 100
	homeNewsLimit    = 3
	homeNewsTimeout  = 4 * time.Second
)

type Home struct {
	mail     mailstore.Store
	calendar calendarstore.Store
	notes    notestore.Store
	drive    drivestore.Store
	tasks    taskstore.Store
	contacts contactstore.Store
	news     NewsFetcher
	theme    theme.Theme
	navigate HomeNavigateFunc

	summary    homeSummary
	cards      []card.Model
	cardIDs    []string
	focusedIdx int
	loaded     bool
	keys       HomeKeyMap
}

type HomeKeyMap struct {
	Refresh    key.Binding
	NextCard   key.Binding
	PrevCard   key.Binding
	OpenCard   key.Binding
	ClearFocus key.Binding
	Mail       key.Binding
	Calendar   key.Binding
	Notes      key.Binding
	Drive      key.Binding
	Contacts   key.Binding
	Tasks      key.Binding
	News       key.Binding
}

type homeSummary struct {
	mail     mailCardData
	calendar calendarCardData
	notes    notesCardData
	drive    driveCardData
	tasks    tasksCardData
	contacts contactsCardData
	news     newsCardData
	lastSync time.Time
}

type mailCardData struct {
	hasMailboxes bool
	unread       int
	drafts       int
	outbox       int
	recent       []mailRecent
	syncedAt     time.Time
	err          error
}

type mailRecent struct {
	subject  string
	from     string
	received time.Time
	unread   bool
}

type calendarCardData struct {
	today    int
	thisWeek int
	upcoming []calendarstore.OccurrenceMeta
	syncedAt time.Time
	err      error
}

type notesCardData struct {
	total    int
	folders  int
	recent   []notesRecent
	syncedAt time.Time
	err      error
}

type notesRecent struct {
	title   string
	updated time.Time
}

type driveCardData struct {
	files    int
	bytes    int64
	recent   []driveRecent
	syncedAt time.Time
	err      error
}

type driveRecent struct {
	name    string
	updated time.Time
}

type tasksCardData struct {
	projects int
	cards    int
	recent   []tasksRecent
	syncedAt time.Time
	err      error
}

type tasksRecent struct {
	title   string
	project string
	updated time.Time
}

type contactsCardData struct {
	total    int
	comms    int
	recent   []contactsRecent
	syncedAt time.Time
	err      error
}

type contactsRecent struct {
	who     string
	subject string
	when    time.Time
}

type newsCardData struct {
	loaded  bool
	recent  []newsRecent
	fetched time.Time
	err     error
}

type newsRecent struct {
	title    string
	score    int
	comments int
	posted   time.Time
}

type homeLoadedMsg struct {
	summary homeSummary
}

type homeNewsLoadedMsg struct {
	news newsCardData
}
