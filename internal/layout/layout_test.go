package layout

import "testing"

func TestCalculateNormalSize(t *testing.T) {
	dims := Calculate(100, 40, true)
	if dims.Header.Height != 2 || dims.Footer.Height != 2 {
		t.Fatalf("unexpected chrome heights: header=%d footer=%d", dims.Header.Height, dims.Footer.Height)
	}
	if dims.Sidebar.Width != sidebarWidth {
		t.Fatalf("unexpected sidebar width: %d", dims.Sidebar.Width)
	}
	if dims.Main.Width != 82 || dims.Main.Height != 36 {
		t.Fatalf("unexpected main size: %dx%d", dims.Main.Width, dims.Main.Height)
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
