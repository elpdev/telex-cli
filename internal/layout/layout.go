package layout

const (
	headerHeight = 2
	footerHeight = 2
	sidebarWidth = 18
)

func Calculate(width, height int, showSidebar bool) Dimensions {
	width = max(0, width)
	height = max(0, height)

	header := min(headerHeight, height)
	footer := 0
	if height > header {
		footer = min(footerHeight, height-header)
	}
	bodyHeight := max(0, height-header-footer)

	sidebar := 0
	if showSidebar && width >= 24 {
		sidebar = min(sidebarWidth, width)
	}

	return Dimensions{
		Width:   width,
		Height:  height,
		Header:  Region{Width: width, Height: header},
		Sidebar: Region{Width: sidebar, Height: bodyHeight},
		Main:    Region{Width: max(0, width-sidebar), Height: bodyHeight},
		Footer:  Region{Width: width, Height: footer},
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
