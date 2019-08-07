package hashcat5

import (
	"testing"
)

var one = []string{"One", "1"}
var two = []string{"Two", "2"}
var three = []string{"Three", "3"}
var four = []string{"Four", "4"}
var five = []string{"Five", "5"}
var six = []string{"Six", "6"}

var OneDuplicateSlice [][]string
var ManyDuplciateSlice [][]string

func TestDedupOneDuplicate(t *testing.T) {
	OneDuplicateSlice = append(OneDuplicateSlice, one, two, three, four, two)
	if len(OneDuplicateSlice) != 5 {
		t.Fatalf("Input length is %d", len(OneDuplicateSlice))
	}
	output := dedupHashes(OneDuplicateSlice)

	if len(output) != 4 {
		t.Fatalf("Output length is %d", len(output))
	}
}

func TestDedupManyDuplicate(t *testing.T) {
	ManyDuplciateSlice = append(OneDuplicateSlice, one, two, three, four, two, six, six, five, four, one, three, two, five, one, one, three)
	if len(ManyDuplciateSlice) != 16 {
		t.Fatalf("Input length is %d", len(ManyDuplciateSlice))
	}
	output := dedupHashes(ManyDuplciateSlice)

	if len(output) != 6 {
		t.Fatalf("Output length is %d", len(output))
	}
}
