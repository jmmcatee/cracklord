package hashcat

type rule struct {
	Name string
	Path string
}

type rules []rule

func (r rules) Len() int {
	return len(r)

}
func (r rules) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r rules) Less(i, j int) bool {
	return r[i].Name < r[j].Name
}
