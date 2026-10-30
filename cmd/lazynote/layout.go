package main

const (
	// notebookHeight is the outer height of the Notebook panel: one content
	// row plus the border.
	notebookHeight = 3
	// titleHeight is the outer height of the Title panel: one content row
	// plus the border.
	titleHeight = 3
	// minSideHeight is the smallest outer height given to Related and Body.
	minSideHeight = 5
	// minInfoHeight is the smallest outer height given to Info, which lists
	// a note's properties and so needs room.
	minInfoHeight = 7
	// leftWidthPercent is the left column share of the terminal width.
	leftWidthPercent = 35
)

// grid holds the outer (border included) size of every panel.
type grid struct {
	leftWidth, rightWidth                     int
	notebookHeight, paraHeight, relatedHeight int
	titleHeight, bodyHeight, infoHeight       int
}

// computeGrid splits width x height into the two-column layout: Notebook,
// PARA and Related on the left; Title, Body and Info, top to bottom, on the
// right. Related takes about a quarter of the height and Info a third, PARA
// and Body take the rest.
func computeGrid(width, height int) grid {
	g := grid{
		leftWidth:      width * leftWidthPercent / 100,
		notebookHeight: notebookHeight,
	}
	g.rightWidth = width - g.leftWidth

	side := max(height/4, minSideHeight)
	g.relatedHeight = side
	g.paraHeight = max(height-g.notebookHeight-g.relatedHeight, minSideHeight)
	g.titleHeight = titleHeight
	g.infoHeight = max(height/3, minInfoHeight)
	g.bodyHeight = max(height-g.titleHeight-g.infoHeight, minSideHeight)
	return g
}
