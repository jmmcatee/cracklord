package nmap

import (
	"encoding/json"
	"fmt"
	"github.com/jmmcatee/cracklord/common"
	"github.com/stretchr/testify/assert"
	"os"
	"strconv"
	"testing"
)

func TestNewTooler(t *testing.T) {
	tooler := NewTooler()
	assert.Implements(t, (*common.Tooler)(nil), tooler)
}

func TestParameters(t *testing.T) {
	var tmp map[string]interface{}
	tmp = make(map[string]interface{})
	tool := NewTooler()
	j := tool.Parameters()
	err := json.Unmarshal([]byte(j), &tmp)
	assert.NoError(t, err, "Unable to parse parameters JSON")
}

func TestGetSortedKeys(t *testing.T) {
	tmp := map[string]string{
		"1": "data",
		"4": "data",
		"3": "data",
		"2": "data",
		"5": "data",
	}
	keys := getSortedKeys(tmp)

	assert.Equal(t, len(tmp), len(keys), "Did not have the same length")

	for i := 1; i < len(tmp); i++ {
		tmpi, _ := strconv.Atoi(keys[i-1])
		assert.Equal(t, i, tmpi)
	}
}

func TestParseNmapXML(t *testing.T) {
	wd, _ := os.Getwd()
	for i := 1; i <= 16; i++ {
		nmap, err := parseNmapXML(fmt.Sprintf("%s/test/xml_test%d.xml", wd, i))

		assert.NoError(t, err, "Unable to parse file %d", i)
		assert.NotEmpty(t, nmap.Hosts, "Unable to parse file %d", i)

		if err == nil {
			csv := nmapToCSV(nmap)
			assert.NotEmpty(t, csv, "There was no output for %d", i)
		}
	}
}
