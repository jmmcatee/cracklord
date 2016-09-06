package hashcat3

type Dictionary struct {
	Name string
	Path string
}

type Dictionaries []Dictionary

func (d Dictionaries) Len() int {
	return len(d)

}
func (d Dictionaries) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func (d Dictionaries) Less(i, j int) bool {
	return d[i].Name < d[j].Name
}
