package screens

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/opener"
)

func (m Mail) handleLinksKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if key.Matches(msg, m.keys.Back) {
		m.mode = mailModeDetail
		return m, nil
	}
	if len(m.links) == 0 {
		return m, nil
	}
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.linkIndex > 0 {
			m.linkIndex--
		}
	case key.Matches(msg, m.keys.Down):
		if m.linkIndex < len(m.links)-1 {
			m.linkIndex++
		}
	case key.Matches(msg, m.keys.Open):
		return m.openLink()
	case key.Matches(msg, m.keys.Copy):
		return m.copyLink()
	case key.Matches(msg, m.keys.Extract):
		return m.extractLink()
	}
	return m, nil
}

func (m Mail) handleArticleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if key.Matches(msg, m.keys.Back) {
		m.mode = mailModeLinks
		return m, nil
	}
	switch {
	case key.Matches(msg, m.keys.Up):
		m.syncArticleViewport(mailReadWidth, 1)
		m.articleViewport.ScrollUp(1)
	case key.Matches(msg, m.keys.Down):
		m.syncArticleViewport(mailReadWidth, 1)
		m.articleViewport.ScrollDown(1)
	case key.Matches(msg, m.keys.Open):
		return m.openArticleURL()
	case key.Matches(msg, m.keys.Copy):
		return m.copyArticleURL()
	}
	return m, nil
}

func (m Mail) openHTML() (Screen, tea.Cmd) {
	if len(m.messages) == 0 {
		return m, nil
	}
	path := filepath.Join(m.messages[m.messageIndex].Path, "body.html")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			m.status = "No cached HTML body for this message"
			return m, nil
		}
		m.status = fmt.Sprintf("Could not read HTML body: %v", err)
		return m, nil
	}
	m.status = "Opening HTML in browser..."
	cmd, err := opener.Command(path)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return htmlOpenFinishedMsg{path: path, err: err}
	})
}

func (m Mail) openLink() (Screen, tea.Cmd) {
	if len(m.links) == 0 {
		return m, nil
	}
	url := m.links[m.linkIndex].URL
	m.status = "Opening link in browser..."
	cmd, err := opener.Command(url)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return linkOpenFinishedMsg{url: url, err: err}
	})
}

func (m Mail) copyLink() (Screen, tea.Cmd) {
	if len(m.links) == 0 {
		return m, nil
	}
	url := m.links[m.linkIndex].URL
	cmd, err := clipboardCommand(url)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.status = "Copying link..."
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return linkCopyFinishedMsg{url: url, err: err}
	})
}

func (m Mail) extractLink() (Screen, tea.Cmd) {
	if len(m.links) == 0 {
		return m, nil
	}
	url := m.links[m.linkIndex].URL
	m.status = "Extracting article..."
	return m, func() tea.Msg {
		article, err := extractArticleURL(context.Background(), url)
		return articleExtractedMsg{url: url, article: article, err: err}
	}
}

func (m Mail) openArticleURL() (Screen, tea.Cmd) {
	if m.articleURL == "" {
		return m, nil
	}
	url := m.articleURL
	m.status = "Opening article in browser..."
	cmd, err := opener.Command(url)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return linkOpenFinishedMsg{url: url, err: err}
	})
}

func (m Mail) copyArticleURL() (Screen, tea.Cmd) {
	if m.articleURL == "" {
		return m, nil
	}
	cmd, err := clipboardCommand(m.articleURL)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.status = "Copying article URL..."
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return linkCopyFinishedMsg{url: m.articleURL, err: err}
	})
}

func clipboardCommand(value string) (*exec.Cmd, error) {
	for _, candidate := range []struct {
		name string
		args []string
	}{
		{name: "wl-copy"},
		{name: "xclip", args: []string{"-selection", "clipboard"}},
		{name: "xsel", args: []string{"--clipboard", "--input"}},
	} {
		if _, err := exec.LookPath(candidate.name); err == nil {
			cmd := exec.Command(candidate.name, candidate.args...)
			cmd.Stdin = strings.NewReader(value)
			return cmd, nil
		}
	}
	return nil, fmt.Errorf("no clipboard command found: install wl-copy, xclip, or xsel")
}
