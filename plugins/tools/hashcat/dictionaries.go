package hashcat

type dictionary struct {
	Name string
	Path string
}

type dictionaries []dictionary

func (d dictionaries) Len() int {
	return len(d)

}
func (d dictionaries) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func (d dictionaries) Less(i, j int) bool {
	return d[i].Name < d[j].Name
}
