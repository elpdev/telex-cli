package commands

import (
	"sort"
	"strings"
)

type Registry struct {
	commands map[string]Command
}

func NewRegistry() *Registry {
	return &Registry{commands: make(map[string]Command)}
}

func (r *Registry) Register(command Command) {
	r.commands[command.ID] = command
}

func (r *Registry) Find(id string) (Command, bool) {
	command, ok := r.commands[id]
	return command, ok
}

func (r *Registry) List() []Command {
	commands := make([]Command, 0, len(r.commands))
	for _, command := range r.commands {
		commands = append(commands, command)
	}
	sort.Slice(commands, func(i, j int) bool { return commands[i].Title < commands[j].Title })
	return commands
}

func (r *Registry) Filter(query string) []Command {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return r.List()
	}

	var matches []Command
	for _, command := range r.List() {
		text := strings.ToLower(command.Title + " " + command.Description + " " + strings.Join(command.Keywords, " "))
		if strings.Contains(text, query) {
			matches = append(matches, command)
		}
	}
	return matches
}
