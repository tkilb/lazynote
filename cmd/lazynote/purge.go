package main

import (
	"fmt"

	"github.com/tkilb/lazynote/internal/ui/popup"
)

// notifyPurge reports the result of the startup purge of archived daily
// notes past the retention period. Deletion is permanent, so any removal is
// reported, and so is a failure.
func (m model) notifyPurge(n int, err error) model {
	if err != nil {
		text := fmt.Sprintf("Could not remove old archived daily notes: %s.", err)
		if n > 0 {
			text = fmt.Sprintf("Removed %d old archived daily %s, but %s.", n, plural(n), err)
		}
		return m.notify(popup.Warning, text)
	}
	if n > 0 {
		return m.notify(popup.Info, fmt.Sprintf("Removed %d archived daily %s older than the retention period.", n, plural(n)))
	}
	return m
}

func plural(n int) string {
	if n == 1 {
		return "note"
	}
	return "notes"
}
