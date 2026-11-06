package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tkilb/para/internal/findtext"
)

func line(rel string, index int, text string) findtext.Line {
	return findtext.Line{Rel: rel, Index: index, Text: text}
}

func TestFindIgnoresCaseAndMarksEveryOccurrence(t *testing.T) {
	lines := []findtext.Line{
		line("a/Alpha.md", 0, "Kanban board, kanban cards"),
		line("a/Alpha.md", 1, "nothing here"),
		line("a/Beta.md", 4, "KANBAN"),
	}
	res := Find(lines, "kan", 10)
	require.Len(t, res.Hits, 2)
	assert.Equal(t, 2, res.Total)
	assert.False(t, res.Truncated)
	assert.Equal(t, "a/Alpha.md", res.Hits[0].Rel)
	assert.Equal(t, 0, res.Hits[0].Index)
	assert.Equal(t, [][2]int{{0, 3}, {14, 17}}, res.Hits[0].Spans)
	assert.Equal(t, 1, res.Hits[0].Col)
	assert.Equal(t, [][2]int{{0, 3}}, res.Hits[1].Spans)
}

func TestFindNeedsEveryWordOnTheLineInAnyOrder(t *testing.T) {
	lines := []findtext.Line{
		line("n.md", 0, "Carolyn: write the investigation card"),
		line("n.md", 1, "write only"),
		line("n.md", 2, "card before write"),
	}
	res := Find(lines, "write CARD", 10)
	require.Len(t, res.Hits, 2)
	assert.Equal(t, 0, res.Hits[0].Index)
	assert.Equal(t, 2, res.Hits[1].Index)
	assert.Equal(t, [][2]int{{9, 14}, {33, 37}}, res.Hits[0].Spans, "spans of both words, in line order")
	assert.Equal(t, 10, res.Hits[0].Col)
}

func TestFindMergesOverlappingSpans(t *testing.T) {
	res := Find([]findtext.Line{line("n.md", 0, "a kanban b")}, "ban kan kanb", 10)
	require.Len(t, res.Hits, 1)
	assert.Equal(t, [][2]int{{2, 8}}, res.Hits[0].Spans)
}

func TestFindColumnCountsCharactersNotBytes(t *testing.T) {
	res := Find([]findtext.Line{line("n.md", 0, "héllo wörld")}, "wörld", 10)
	require.Len(t, res.Hits, 1)
	assert.Equal(t, 7, res.Hits[0].Col)
	assert.Equal(t, [][2]int{{7, 13}}, res.Hits[0].Spans, "spans are byte offsets")
}

func TestFindFoldsCaseBeyondAscii(t *testing.T) {
	res := Find([]findtext.Line{line("n.md", 0, "the ÅNGSTRÖM unit")}, "ångström", 10)
	require.Len(t, res.Hits, 1)
	assert.Equal(t, [][2]int{{4, 14}}, res.Hits[0].Spans)
}

func TestFindWithNothingTypedFindsNothing(t *testing.T) {
	lines := []findtext.Line{line("n.md", 0, "text")}
	for _, q := range []string{"", "   ", "\t"} {
		res := Find(lines, q, 10)
		assert.Empty(t, res.Hits, "%q", q)
		assert.Zero(t, res.Total)
	}
}

func TestFindPutsNotesWhoseNameMatchesFirst(t *testing.T) {
	lines := []findtext.Line{
		line("areas/Misc.md", 0, "talk to the CTO"),
		line("areas/CTO Etiquette.md", 3, "cto rules"),
		line("areas/CTO Etiquette.md", 5, "ask the cto"),
		line("areas/Other.md", 1, "cto again"),
	}
	res := Find(lines, "cto", 10)
	require.Len(t, res.Hits, 4)
	assert.Equal(t, "areas/CTO Etiquette.md", res.Hits[0].Rel)
	assert.Equal(t, "areas/CTO Etiquette.md", res.Hits[1].Rel)
	assert.Equal(t, "areas/Misc.md", res.Hits[2].Rel, "then the rest in note order")
	assert.Equal(t, "areas/Other.md", res.Hits[3].Rel)
}

func TestFindLimitsHitsButCountsAll(t *testing.T) {
	var lines []findtext.Line
	for i := 0; i < 50; i++ {
		lines = append(lines, line("n.md", i, "match"))
	}
	res := Find(lines, "match", 20)
	assert.Len(t, res.Hits, 20)
	assert.Equal(t, 50, res.Total)
	assert.True(t, res.Truncated)
	assert.Equal(t, 0, res.Hits[0].Index)
}

func TestFindKeepsNameMatchesWhenTheLimitIsReached(t *testing.T) {
	var lines []findtext.Line
	for i := 0; i < 30; i++ {
		lines = append(lines, line("zeta.md", i, "cto"))
	}
	lines = append(lines, line("CTO.md", 0, "cto"))
	res := Find(lines, "cto", 5)
	require.Len(t, res.Hits, 5)
	assert.Equal(t, "CTO.md", res.Hits[0].Rel, "a note named for the query is never pushed out by the limit")
}
