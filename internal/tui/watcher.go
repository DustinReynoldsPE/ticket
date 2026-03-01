package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

type fileChangedMsg struct{}

// watchTickets watches the tickets directory for changes and sends
// fileChangedMsg to the program. Events are debounced to avoid
// redundant reloads from rapid writes.
func watchTickets(dir string) tea.Cmd {
	return func() tea.Msg {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return nil
		}
		if err := watcher.Add(dir); err != nil {
			watcher.Close()
			return nil
		}

		// Wait for the first event, then debounce.
		for {
			select {
			case _, ok := <-watcher.Events:
				if !ok {
					return nil
				}
				// Drain events for debounce window.
				timer := time.NewTimer(200 * time.Millisecond)
			drain:
				for {
					select {
					case _, ok := <-watcher.Events:
						if !ok {
							timer.Stop()
							return nil
						}
					case <-timer.C:
						break drain
					}
				}
				watcher.Close()
				return fileChangedMsg{}
			case _, ok := <-watcher.Errors:
				if !ok {
					return nil
				}
			}
		}
	}
}
