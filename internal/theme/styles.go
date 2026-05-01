package theme

import "github.com/elpdev/tuitheme"

func BuiltIns() []Theme {
	return tuitheme.BuiltIns()
}

func Next(current string) Theme {
	return tuitheme.Next(BuiltIns(), current)
}

func Phosphor() Theme {
	return tuitheme.Phosphor()
}
