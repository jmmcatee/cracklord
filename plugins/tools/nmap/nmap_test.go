package nmap

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/jmmcatee/cracklord/common"
	"github.com/stretchr/testify/assert"
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

func TestGetCIDRTargetCount(t *testing.T) {
	test := map[string]int{
		"192.168.1.0/24":  256,
		"10.0.0.0/8":      16777216,
		"67.52.98.20/28":  16,
		"172.16.14.72/30": 4,
	}

	for r, v := range test {
		count, err := getCIDRTargetCount(r)
		assert.NoError(t, err, "Unable to get CIDR range address count")
		assert.Equal(t, v, count, "CIDR ranges did not match")
	}
}

func TestGetRangeTargetCount(t *testing.T) {
	test := map[string]int{
		"192.168.1.1-255":         255,
		"10.0.1-255.1-255":        65025,
		"1-4.1-4.1-4.1-4":         256,
		"65-67.1-255.1-255.1-255": 49744125,
	}

	for r, v := range test {
		count, err := getRangeTargetCount(r)
		assert.NoError(t, err, "Unable to get CIDR range address count")
		assert.Equal(t, v, count, "CIDR ranges did not match")
	}
}

func TestCalcTotalTargets(t *testing.T) {
	data := `192.168.1.0/24
10.0.1-255.1-255
192.168.1.1`

	count, err := calcTotalTargets(data)
	assert.NoError(t, err, "Unable to get total count")
	assert.Equal(t, int64(65282), count)
}
