package hashcat

import (
	"testing"
)

func TestHashesPerSecondParsing(t *testing.T) {
	var testStringMH = `
Speed.GPU.#1...:   129.0 MH/s
Speed.GPU.#2...:   129.0 MH/s
Speed.GPU.#3...:   130.3 MH/s
Speed.GPU.#4...:   131.1 MH/s
Speed.GPU.#5...:   128.4 MH/s
Speed.GPU.#6...:   129.6 MH/s
Speed.GPU.#7...:   128.6 MH/s
Speed.GPU.#8...:   128.4 MH/s
Speed.GPU.#*...:  1035.5 MH/s`

	var testStringkH = `
Session.Name...: a5449832-6c32-4f1c-b593-c1facaac8afe
Status.........: Running
Rules.Type.....: File (/var/cracklord/oclHashcat/rules/d3ad0ne.rule)
Input.Mode.....: File (/var/cracklord/oclHashcat/dicts/crackstation-human-only.txt)
Hash.Target....: $DCC2$10240#tom#e4e938d12fe5974dc42b90120...
Hash.Type......: DCC2, mscash2
Time.Started...: Thu Jul  2 13:05:51 2015 (20 secs)
Time.Estimated.: Mon Aug 17 16:55:35 2015 (46 days, 3 hours)
Speed.GPU.#1...:    69970 H/s
Speed.GPU.#2...:    70069 H/s
Speed.GPU.#3...:    70005 H/s
Speed.GPU.#4...:    69934 H/s
Speed.GPU.#5...:    69841 H/s
Speed.GPU.#6...:    69789 H/s
Speed.GPU.#7...:    69894 H/s
Speed.GPU.#8...:    69895 H/s
Speed.GPU.#*...:   559.4 kH/s`

	parsedStringSlice := regGPUSpeed.FindAllStringSubmatch(testStringkH, -1)

	if len(parsedStringSlice) != 9 {
		t.Errorf("Parsing is not correct for H/s and/or kH/s. Length of parse was %d.\n", len(parsedStringSlice))
	}

	parsedStringSlice = regGPUSpeed.FindAllStringSubmatch(testStringMH, -1)

	if len(parsedStringSlice) != 9 {
		t.Errorf("Parsing is not correct for MH/s. Length of parse was %d.\n", len(parsedStringSlice))
	}
}
