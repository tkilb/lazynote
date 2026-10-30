package store

import (
	"path"
	"strings"
	"time"
)

const (
	tabInbox    = "inbox"
	tabArchives = "archives"
	dailyDir    = "daily"
)

// tabRoots are the folders PARA shows as tabs.
var tabRoots = []string{tabInbox, "projects", "areas", "resources", tabArchives}

// MoveDestination returns where the note at rel goes when moved to the tab
// folder tab: the root of that tab, keeping the file name. A dated daily note
// (inbox/daily/YYYY-MM-DD.md) moved to the archives goes to archives/daily
// instead, so the retention purge applies to it. It returns "" for an
// unknown tab or an empty rel.
func MoveDestination(rel, tab string) string {
	known := false
	for _, t := range tabRoots {
		known = known || t == tab
	}
	if !known || rel == "" {
		return ""
	}
	name := path.Base(rel)
	if tab == tabArchives && path.Dir(rel) == path.Join(tabInbox, dailyDir) && isDailyName(name) {
		return path.Join(tabArchives, dailyDir, name)
	}
	return path.Join(tab, name)
}

// isDailyName reports whether name is YYYY-MM-DD.md, ignoring the case of
// the extension.
func isDailyName(name string) bool {
	if !isNote(name) {
		return false
	}
	_, err := time.Parse(dateLayout, strings.TrimSuffix(name, name[len(name)-len(noteExt):]))
	return err == nil
}

// DailyDate returns the date of a dated daily note: a note named
// YYYY-MM-DD.md directly inside inbox/daily or archives/daily. It reports
// false for anything else.
func DailyDate(rel string) (string, bool) {
	dir := path.Dir(rel)
	if dir != path.Join(tabInbox, dailyDir) && dir != path.Join(tabArchives, dailyDir) {
		return "", false
	}
	name := path.Base(rel)
	if !isDailyName(name) {
		return "", false
	}
	return name[:len(name)-len(noteExt)], true
}
