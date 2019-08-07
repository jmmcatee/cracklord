package hashcat5

type RuleFile struct {
	Name string
	Path string
}

type RuleFiles []RuleFile

func (r RuleFiles) Len() int {
	return len(r)

}
func (r RuleFiles) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r RuleFiles) Less(i, j int) bool {
	return r[i].Name < r[j].Name
}
