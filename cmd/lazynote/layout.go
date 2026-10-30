package main

const (
	// notebookHeight is the outer height of the Notebook panel: one content
	// row plus the border.
	notebookHeight = 3
	// minSideHeight is the smallest outer height given to Related and Info.
	minSideHeight = 5
	// leftWidthPercent is the left column share of the terminal width.
	leftWidthPercent = 35
)

// grid holds the outer (border included) size of every panel.
type grid struct {
	leftWidth, rightWidth                     int
	notebookHeight, paraHeight, relatedHeight int
	infoHeight, bodyHeight                    int
}

// computeGrid splits width x height into the two-column layout: Notebook,
// PARA and Related on the left; Info and Body on the right. Related and Info
// take about a quarter of the height, PARA and Body take the rest.
func computeGrid(width, height int) grid {
	g := grid{
		leftWidth:      width * leftWidthPercent / 100,
		notebookHeight: notebookHeight,
	}
	g.rightWidth = width - g.leftWidth

	side := max(height/4, minSideHeight)
	g.relatedHeight = side
	g.paraHeight = max(height-g.notebookHeight-g.relatedHeight, minSideHeight)
	g.infoHeight = side
	g.bodyHeight = max(height-g.infoHeight, minSideHeight)
	return g
}
