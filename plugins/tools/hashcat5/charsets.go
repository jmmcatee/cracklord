package hashcat5

// CharSetPreDefCustom1 is a custom character set that includes lower case and numbers
var CharSetPreDefCustom1 = "?l?d"

// CharSetPreDefCustom2 is a custom character set that includes lower case, upper case and numbers
var CharSetPreDefCustom2 = "?u?l?d"

// CharSetPreDefCustom3 is a custom character set that includes numbers and symbols
var CharSetPreDefCustom3 = "?d?s"

// CharSetPreDefCustom4 is a custom character set that includes lower case, numbers and symbols
var CharSetPreDefCustom4 = "?l?d?s"

// Charset is a structure representing a character set for Hashcat. It has a name for easy of selection and a Mask that is the value given to Hashcat.
type Charset struct {
	Name string
	Mask string
}

// Charsets is a slice of Charset structs
type Charsets []Charset

// Len is the length of the Charsets for sorting
func (r Charsets) Len() int {
	return len(r)

}

// Swap is the swap function of the Charsets for sorting
func (r Charsets) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

// Less is the comparison of the Charsets for sorting
func (r Charsets) Less(i, j int) bool {
	return r[i].Name < r[j].Name
}
