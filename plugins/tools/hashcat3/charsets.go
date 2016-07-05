package hashcat3

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
