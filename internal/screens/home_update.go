package screens

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func (h Home) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case homeLoadedMsg:
		// Preserve any already-loaded news data so a refresh of the main
		// summary doesn't clobber a successful news fetch from earlier.
		news := h.summary.news
		h.summary = msg.summary
		if news.loaded {
			h.summary.news = news
		}
		h.loaded = true
		h.cards, h.cardIDs = h.buildCards()
		return h, nil
	case homeNewsLoadedMsg:
		h.summary.news = msg.news
		h.cards, h.cardIDs = h.buildCards()
		return h, nil
	case tea.KeyPressMsg:
		return h.handleKey(msg)
	}
	return h, nil
}

func (h Home) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch {
	case key.Matches(msg, h.keys.Refresh):
		return h, h.loadCmd()
	case key.Matches(msg, h.keys.Mail):
		return h.routeTo("mail")
	case key.Matches(msg, h.keys.Calendar):
		return h.routeTo("calendar")
	case key.Matches(msg, h.keys.Contacts):
		return h.routeTo("contacts")
	case key.Matches(msg, h.keys.Notes):
		return h.routeTo("notes")
	case key.Matches(msg, h.keys.Tasks):
		return h.routeTo("tasks")
	case key.Matches(msg, h.keys.Drive):
		return h.routeTo("drive")
	case key.Matches(msg, h.keys.News):
		return h.routeTo("news")
	case key.Matches(msg, h.keys.NextCard):
		return h.moveFocus(1), nil
	case key.Matches(msg, h.keys.PrevCard):
		return h.moveFocus(-1), nil
	case key.Matches(msg, h.keys.ClearFocus):
		if h.focusedIdx >= 0 {
			return h.setFocus(-1), nil
		}
	case key.Matches(msg, h.keys.OpenCard):
		if h.focusedIdx >= 0 && h.focusedIdx < len(h.cardIDs) {
			return h.routeTo(h.cardIDs[h.focusedIdx])
		}
	}
	return h, nil
}

func (h Home) routeTo(id string) (Screen, tea.Cmd) {
	if h.navigate == nil {
		return h, nil
	}
	return h, h.navigate(id)
}

func (h Home) moveFocus(delta int) Home {
	if len(h.cards) == 0 {
		return h
	}
	next := h.focusedIdx + delta
	if h.focusedIdx < 0 {
		if delta > 0 {
			next = 0
		} else {
			next = len(h.cards) - 1
		}
	}
	if next < 0 {
		next = len(h.cards) - 1
	}
	if next >= len(h.cards) {
		next = 0
	}
	return h.setFocus(next)
}

func (h Home) setFocus(idx int) Home {
	h.focusedIdx = idx
	for i := range h.cards {
		if i == idx {
			h.cards[i] = h.cards[i].Focus()
		} else {
			h.cards[i] = h.cards[i].Blur()
		}
	}
	return h
}
