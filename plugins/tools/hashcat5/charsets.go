package hashcat5

// Assumes ?1=?l?d, ?2=?u?l?d, ?3=?d?s, ?4=?l?d?s
var CharSetPreDefCustom1 = "?l?d"   // Lower ad number
var CharSetPreDefCustom2 = "?u?l?d" // Upper, lower, and number
var CharSetPreDefCustom3 = "?d?s"   // number and symbols
var CharSetPreDefCustom4 = "?l?d?s" // lower, number, and symbol

type Charset struct {
	Name string
	Mask string
}

type Charsets []Charset

func (r Charsets) Len() int {
	return len(r)

}
func (r Charsets) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r Charsets) Less(i, j int) bool {
	return r[i].Name < r[j].Name
}
