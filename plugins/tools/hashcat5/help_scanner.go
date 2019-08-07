package hashcat5

import (
	"bufio"
	"strings"
)

// HashcatHelpScanner is designed to return a table struct of the hashcat help output
func HashcatHelpScanner(help string, section string) map[string][]string {
	table := map[string][]string{}
	scanner := bufio.NewScanner(strings.NewReader(help))

	// Loop through the lines looking for the section header provided
	for scanner.Scan() {
		if strings.Compare(scanner.Text(), "- [ "+section+" ] -") == 0 {
			println(scanner.Text() + "\n")
			// We found our section, read the next empty line and then take in the tables
			scanner.Scan() // empty line
			scanner.Scan() // Should be our table header

			// Split up the table headers on the '|' character
			headers := strings.Split(scanner.Text(), "|")
			headermap := map[int]string{}
			for i, colHeader := range headers {
				key := strings.TrimSpace(colHeader)
				headermap[i] = key
				table[key] = []string{}
			}

			// Skip the header seperator
			scanner.Scan()

			// Now start looping on table values looking for an empty line
			for scanner.Scan() {
				if scanner.Text() == "" {
					break
				}

				values := strings.Split(scanner.Text(), " | ")
				for i, value := range values {
					table[headermap[i]] = append(table[headermap[i]], strings.TrimSpace(value))
				}
			}

			break
		}
	}

	return table
}
