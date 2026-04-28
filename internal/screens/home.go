package screens

import (
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/calendarstore"
	"github.com/elpdev/telex-cli/internal/contactstore"
	"github.com/elpdev/telex-cli/internal/drivestore"
	"github.com/elpdev/telex-cli/internal/mailstore"
	"github.com/elpdev/telex-cli/internal/notestore"
	"github.com/elpdev/telex-cli/internal/taskstore"
	"github.com/elpdev/telex-cli/internal/theme"
)

func NewHome(
	mail mailstore.Store,
	calendar calendarstore.Store,
	notes notestore.Store,
	drive drivestore.Store,
	tasks taskstore.Store,
	contacts contactstore.Store,
	news NewsFetcher,
	t theme.Theme,
	navigate HomeNavigateFunc,
) Home {
	h := Home{
		mail:       mail,
		calendar:   calendar,
		notes:      notes,
		drive:      drive,
		tasks:      tasks,
		contacts:   contacts,
		news:       news,
		theme:      t,
		navigate:   navigate,
		focusedIdx: -1,
		keys:       DefaultHomeKeyMap(),
	}
	h.cards, h.cardIDs = h.buildCards()
	return h
}

func (h Home) Reconfigure(t theme.Theme) Home {
	h.theme = t
	h.cards, h.cardIDs = h.buildCards()
	return h
}

func (h Home) Init() tea.Cmd { return h.loadCmd() }

func (h Home) Title() string { return "Home" }
