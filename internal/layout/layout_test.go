package layout

import "testing"

func TestCalculateNormalSize(t *testing.T) {
	dims := Calculate(100, 40, true)
	if dims.Header.Height != bannerHeaderHeight || dims.Footer.Height != footerHeight {
		t.Fatalf("unexpected chrome heights: header=%d footer=%d", dims.Header.Height, dims.Footer.Height)
	}
	if dims.Sidebar.Width != sidebarWidth {
		t.Fatalf("unexpected sidebar width: %d", dims.Sidebar.Width)
	}
	if dims.Main.Width != 82 || dims.Main.Height != 34 {
		t.Fatalf("unexpected main size: %dx%d", dims.Main.Width, dims.Main.Height)
	}
}

func TestCalculateCompactHeaderOnSmallHeight(t *testing.T) {
	dims := Calculate(100, 10, true)
	if dims.Header.Height != compactHeaderHeight {
		t.Fatalf("expected compact header on short terminal, got %d", dims.Header.Height)
	}
}

func TestCalculateTinySizeNoNegativeDimensions(t *testing.T) {
	dims := Calculate(4, 2, true)
	regions := []Region{dims.Header, dims.Sidebar, dims.Main, dims.Footer}
	for _, region := range regions {
		if region.Width < 0 || region.Height < 0 {
			t.Fatalf("negative region: %+v", region)
		}
	}
}
