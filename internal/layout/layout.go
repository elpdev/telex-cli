package layout

import "github.com/elpdev/tuilayout"

const (
	compactHeaderHeight = 2
	bannerHeaderHeight  = 4
	bannerMinWidth      = 48
	bannerMinHeight     = 14
	footerHeight        = 2
	sidebarWidth        = 18
)

func Calculate(width, height int, showSidebar bool) Dimensions {
	return tuilayout.Calculate(width, height, tuilayout.Options{
		ShowSidebar:  showSidebar,
		HeaderHeight: compactHeaderHeight,
		FooterHeight: footerHeight,
		SidebarWidth: sidebarWidth,
		ResponsiveHeader: tuilayout.ResponsiveHeader{
			Height:    bannerHeaderHeight,
			MinWidth:  bannerMinWidth,
			MinHeight: bannerMinHeight,
		},
	})
}
