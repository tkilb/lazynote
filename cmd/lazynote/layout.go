package main

const (
	// panelBorderRows is the rows a panel's top and bottom borders take.
	panelBorderRows = 2
	// titleHeight is the outer height of the Title panel: one content row
	// plus the border.
	titleHeight = 3
	// minSideHeight is the smallest outer height given to Related and Body.
	minSideHeight = 5
	// infoHeight is the outer height of the Info panel: the three standard
	// rows plus the border. Other properties scroll.
	infoHeight = 5
	// leftWidthPercent is the left column share of the terminal width.
	leftWidthPercent = 35
)

// grid holds the outer (border included) size of every panel.
type grid struct {
	leftWidth, rightWidth                     int
	stickiesHeight, paraHeight, relatedHeight int
	titleHeight, bodyHeight, infoHeight       int
}

// computeGrid splits width x height into the two-column layout: Stickies (of
// outer height stickiesHeight, which grows while it has focus), PARA and
// Related on the left; Title, Body and Info, top to bottom, on the
// right. Related takes about a quarter of the height and Info a fixed five rows, PARA
// and Body take the rest.
func computeGrid(width, height, stickiesHeight int) grid {
	g := grid{
		leftWidth:      width * leftWidthPercent / 100,
		stickiesHeight: stickiesHeight,
	}
	g.rightWidth = width - g.leftWidth

	side := max(height/4, minSideHeight)
	g.relatedHeight = side
	g.paraHeight = max(height-g.stickiesHeight-g.relatedHeight, minSideHeight)
	g.titleHeight = titleHeight
	g.infoHeight = infoHeight
	g.bodyHeight = max(height-g.titleHeight-g.infoHeight, minSideHeight)
	return g
}
