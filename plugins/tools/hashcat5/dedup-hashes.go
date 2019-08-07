package hashcat5

// dedupHashes takes a 2D array of hashes and deduplicates each line so we only return it once.
func dedupHashes(input [][]string) [][]string {
	var deduped [][]string
	dedupMap := make(map[string]bool)
	// Loop on the top level so we can compare rows

	for i := range input {
		// concat the row
		var row string
		for _, v := range input[i] {
			row += v
		}

		if _, value := dedupMap[row]; !value {
			dedupMap[row] = true
			deduped = append(deduped, input[i])
		}
	}

	return deduped
}
