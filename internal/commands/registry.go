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
	if command.Module == "" {
		command.Module = ModuleGlobal
	}
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
	sort.Slice(commands, func(i, j int) bool {
		if commands[i].ID == "quit" || commands[j].ID == "quit" {
			return commands[j].ID == "quit"
		}
		return commands[i].Title < commands[j].Title
	})
	return commands
}

// Scope is the parsed prefix the user typed (e.g. "drafts " -> {Group: "drafts"}).
type Scope struct {
	Module string
	Group  string
}

func (s Scope) IsEmpty() bool { return s.Module == "" && s.Group == "" }

// ParseScope strips a recognised module/group token (followed by a space) from the
// front of the query and returns the remainder. The returned query has the
// matched prefix removed and is left-trimmed.
func ParseScope(query string) (Scope, string) {
	q := strings.ToLower(strings.TrimLeft(query, " "))
	scope := Scope{}
	for {
		token, rest, ok := splitFirstToken(q)
		if !ok {
			break
		}
		if module, isModule := matchModule(token); isModule && scope.Module == "" {
			scope.Module = module
			q = rest
			continue
		}
		if group, isGroup := matchGroup(token); isGroup && scope.Group == "" {
			scope.Group = group
			q = rest
			continue
		}
		break
	}
	return scope, q
}

func matchModule(token string) (string, bool) {
	switch token {
	case ModuleMail, ModuleCalendar, ModuleDrive, ModuleNotes, ModuleHackerNews, ModuleSettings, ModuleGlobal:
		return token, true
	}
	return "", false
}

func matchGroup(token string) (string, bool) {
	switch token {
	case GroupDrafts, GroupMessages, GroupOutbox, GroupInbox:
		return token, true
	}
	return "", false
}

// splitFirstToken returns the first whitespace-delimited token plus the rest of
// the string with leading whitespace removed. ok is false if no trailing space
// was found — this prevents prefix-filter activation while the user is still
// typing the token.
func splitFirstToken(s string) (string, string, bool) {
	idx := strings.IndexByte(s, ' ')
	if idx < 0 {
		return "", "", false
	}
	return s[:idx], strings.TrimLeft(s[idx+1:], " "), true
}

// Filter returns commands matching query in the given context. Recognised
// scope prefixes ("drafts ", "mail ") narrow the result; remaining
// tokens fuzzy-match Title/Description/Keywords. Results are ranked with
// context-module commands first.
func (r *Registry) Filter(query string, ctx Context) []Command {
	scope, rest := ParseScope(query)
	rest = strings.TrimSpace(rest)

	type ranked struct {
		cmd     Command
		bucket  int // 0 = context match, 1 = other
		ordinal int
	}

	all := r.List()
	matches := make([]ranked, 0, len(all))
	for i, cmd := range all {
		if !cmd.IsAvailable(ctx) {
			continue
		}
		if scope.Module != "" && cmd.Module != scope.Module {
			continue
		}
		if scope.Group != "" && cmd.Group != scope.Group {
			continue
		}
		if rest != "" && !textMatch(cmd, rest) {
			continue
		}
		bucket := 1
		if ctx.ActiveScreen != "" && cmd.Module == ctx.ActiveScreen {
			bucket = 0
		}
		matches = append(matches, ranked{cmd: cmd, bucket: bucket, ordinal: i})
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].bucket != matches[j].bucket {
			return matches[i].bucket < matches[j].bucket
		}
		return matches[i].ordinal < matches[j].ordinal
	})

	result := make([]Command, 0, len(matches))
	for _, m := range matches {
		result = append(result, m.cmd)
	}
	return result
}

func textMatch(cmd Command, query string) bool {
	q := strings.ToLower(query)
	hay := strings.ToLower(cmd.Title + " " + cmd.Description + " " + cmd.Module + " " + cmd.Group + " " + strings.Join(cmd.Keywords, " "))
	for _, term := range strings.Fields(q) {
		if !strings.Contains(hay, term) {
			return false
		}
	}
	return true
}

// GroupByModule returns commands grouped by module in display order. The
// active-screen module appears first, followed by the rest in canonical order.
func (r *Registry) GroupByModule(ctx Context) []ModuleGroup {
	canonical := []string{ModuleMail, ModuleCalendar, ModuleDrive, ModuleNotes, ModuleHackerNews, ModuleSettings, ModuleGlobal}
	if ctx.ActiveScreen != "" {
		canonical = bringFirst(canonical, ctx.ActiveScreen)
	}
	buckets := make(map[string][]Command)
	for _, cmd := range r.List() {
		if !cmd.IsAvailable(ctx) {
			continue
		}
		buckets[cmd.Module] = append(buckets[cmd.Module], cmd)
	}
	groups := make([]ModuleGroup, 0, len(canonical))
	for _, module := range canonical {
		groups = append(groups, ModuleGroup{Module: module, Commands: buckets[module]})
	}
	return groups
}

type ModuleGroup struct {
	Module   string
	Commands []Command
}

func bringFirst(values []string, first string) []string {
	out := make([]string, 0, len(values))
	out = append(out, first)
	for _, v := range values {
		if v != first {
			out = append(out, v)
		}
	}
	return out
}
