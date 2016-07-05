package hashcat3

import ()

import (
	"strconv"
)

// HashMode is a structure to hold an instance of hashcat's hash mode data
type HashMode struct {
	Number   string
	Name     string
	Category string
}

// HashModes is a slice of HashMode
type HashModes []HashMode

func (h HashModes) Len() int {
	return len(h)

}
func (h HashModes) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h HashModes) Less(i, j int) bool {
	iNum, _ := strconv.Atoi(h[i].Number)
	jNum, _ := strconv.Atoi(h[j].Number)
	return iNum < jNum
}
