package hashcat5

// Dictionary is a structure for working with dictionaries related to Hashcat. The Dictionary has a name for display and a file path which is given to Hashcat.
type Dictionary struct {
	Name string
	Path string
}

// Dictionaries is a slice of Dictionary structs
type Dictionaries []Dictionary

// Len is a function giving the length of the slice for sorting
func (d Dictionaries) Len() int {
	return len(d)

}

// Swap is a function for swaping slice indexes for sorting
func (d Dictionaries) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

// Less is a comparision function for sorting
func (d Dictionaries) Less(i, j int) bool {
	return d[i].Name < d[j].Name
}
