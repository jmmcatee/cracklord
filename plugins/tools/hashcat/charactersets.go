package hashcat

type characterset struct {
	Name string
	Mask string
}

type charactersets []characterset

func (r charactersets) Len() int {
	return len(r)

}
func (r charactersets) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r charactersets) Less(i, j int) bool {
	return r[i].Name < r[j].Name
}
