package layout

type Region struct {
	Width  int
	Height int
}

type Dimensions struct {
	Width   int
	Height  int
	Header  Region
	Sidebar Region
	Main    Region
	Footer  Region
}
