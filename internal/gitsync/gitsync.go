// Package gitsync reports how far a notebook's git branch is from its
// upstream, or without one how many files are uncommitted, for the status bar. It only reads local state: it never fetches or
// pushes, so "behind" is as of the last fetch.
package gitsync

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// timeout bounds the git call so a stuck repository cannot hold the screen.
const timeout = 3 * time.Second

// Status is how a notebook's repository stands. Uncommitted is the number of
// files in the notebook folder that are changed, added or removed since the
// last commit, untracked ones included. With an upstream (Tracked), Ahead and
// Behind count the commits to push and to pull.
type Status struct {
	Tracked       bool
	Ahead, Behind int
	Uncommitted   int
}

// InSync reports whether a tracked branch has nothing to push or pull.
func (s Status) InSync() bool { return s.Tracked && s.Ahead == 0 && s.Behind == 0 }

// Read returns the status of the repository holding the folder root. It
// reports false when root is not in a repository or git cannot run. A branch
// without an upstream (or without commits) is not Tracked.
func Read(ctx context.Context, root string) (Status, bool) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var st Status
	out, err := exec.CommandContext(ctx, "git", "-C", root,
		"rev-list", "--left-right", "--count", "HEAD...@{upstream}").Output()
	if err == nil {
		st, _ = parse(string(out))
	}
	out, statusErr := exec.CommandContext(ctx, "git", "-C", root,
		"status", "--porcelain", "--untracked-files=all", "--", ".").Output()
	if statusErr != nil {
		return st, st.Tracked
	}
	st.Uncommitted = countLines(string(out))
	return st, true
}

// parse reads the "ahead<TAB>behind" line of git rev-list --left-right --count.
func parse(out string) (Status, bool) {
	fields := strings.Fields(out)
	if len(fields) != 2 {
		return Status{}, false
	}
	ahead, errA := strconv.Atoi(fields[0])
	behind, errB := strconv.Atoi(fields[1])
	if errA != nil || errB != nil || ahead < 0 || behind < 0 {
		return Status{}, false
	}
	return Status{Tracked: true, Ahead: ahead, Behind: behind}, true
}

// countLines counts the non-empty lines of git status --porcelain output,
// one per file.
func countLines(out string) int {
	n := 0
	for _, l := range strings.Split(out, "\n") {
		if strings.TrimSpace(l) != "" {
			n++
		}
	}
	return n
}
