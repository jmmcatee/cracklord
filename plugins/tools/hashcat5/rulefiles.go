package hashcat5

// RuleFile is a structure for storing rules for hashcat. Name is a display name and Path specifies the file path.
type RuleFile struct {
	Name string
	Path string
}

// RuleFiles is a slice of RuleFile structs
type RuleFiles []RuleFile

// Len returns the slice length for the sorting interface.
func (r RuleFiles) Len() int {
	return len(r)

}

// Swap swaps values by index for the sorting interface.
func (r RuleFiles) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

// Less compares values by index for the sorting interface.
func (r RuleFiles) Less(i, j int) bool {
	return r[i].Name < r[j].Name
}
