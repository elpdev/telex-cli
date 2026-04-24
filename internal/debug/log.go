package debug

import "time"

type Entry struct {
	Time    time.Time
	Level   string
	Message string
}

type Log struct {
	entries []Entry
}

func NewLog() *Log { return &Log{} }

func (l *Log) Append(level, message string) {
	l.entries = append(l.entries, Entry{Time: time.Now(), Level: level, Message: message})
}

func (l *Log) Info(message string) { l.Append("INFO", message) }

func (l *Log) Warn(message string) { l.Append("WARN", message) }

func (l *Log) Entries() []Entry {
	entries := make([]Entry, len(l.entries))
	copy(entries, l.entries)
	return entries
}
