package nmap

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/jmmcatee/cracklord/common"
	"github.com/vaughan0/go-ini"
	"sort"
)

type nmapConfig struct {
	BinPath   string
	WorkDir   string
	Arguments string
}

var config = nmapConfig{
	BinPath:   "",
	WorkDir:   "",
	Arguments: "",
}

var scanTypes = map[string]string{
	"ICMP (sP)":             "-sP",
	"TCP SYN (sS)":          "-sS",
	"TCP Connect (sT)":      "-sT",
	"TCP ACK (sA)":          "-sA",
	"TCP Window (sW)":       "-sW",
	"TCP Maimon (sM)":       "-sM",
	"UDP (sU)":              "-sU",
	"TCP Null (sN)":         "-sN",
	"TCP FIN (sF)":          "-sF",
	"TCP Xmas (sX)":         "-sX",
	"SCTP INIT (sY)":        "-sY",
	"SCTP COOKIE-ECHO (sZ)": "-sZ",
}

var timingSettings = map[string]string{
	"Paranoid (0)":   "-T0",
	"Sneaky (1)":     "-T1",
	"Polite (2)":     "-T2",
	"Normal (3)":     "-T3",
	"Aggressive (4)": "-T4",
	"Insane (5)":     "-T5",
}
var timingOrder [6]string

func init() {
	timingOrder[0] = "Paranoid (0)"
	timingOrder[1] = "Sneaky (1)"
	timingOrder[2] = "Polite (2)"
	timingOrder[3] = "Normal (3)"
	timingOrder[4] = "Aggressive (4)"
	timingOrder[5] = "Insane (5)"
}

var portSettings = map[string]string{
	"Custom Port Listing": "",
}

func Setup(path string) error {
	log.Debug("Setting up nmap tool")
	// Join the path provided
	confFile, err := ini.LoadFile(path)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
			"file":  path,
		}).Error("Unable to load configuration file.")
		return err
	}

	// Get the bin path
	basic := confFile.Section("Basic")
	if len(basic) == 0 {
		// Nothing retrieved, so return error
		return errors.New("No \"Basic\" configuration section.")
	}
	config.BinPath = basic["binPath"]
	config.WorkDir = basic["workingdir"]
	config.Arguments = basic["arguments"]

	log.WithFields(log.Fields{
		"binpath":   config.BinPath,
		"WorkDir":   config.WorkDir,
		"Arguments": config.Arguments,
	}).Debug("Basic configuration complete")

	// Get the dictionary section
	portrules := confFile.Section("PortRules")
	if len(portrules) == 0 {
		// Nothing retrieved, so return error
		log.Error("No 'portrules' configuration section in nmap config.")
		return errors.New("No portrules configuration section was found in the nmap configuration.")
	}
	for key, value := range portrules {
		log.WithFields(log.Fields{
			"name":  key,
			"ports": value,
		}).Debug("Added port rule")
		portSettings[key] = value
	}

	log.Info("NMap tool successfully setup")

	return nil
}

type nmapTooler struct {
	toolUUID string
}

func (this *nmapTooler) Name() string {
	return "NMap (Network Mapper) Scan"
}

func (this *nmapTooler) Type() string {
	return "Network Scan"
}

func (this *nmapTooler) Version() string {
	return "6.49"
}

func (this *nmapTooler) UUID() string {
	return this.toolUUID
}

func (this *nmapTooler) SetUUID(s string) {
	this.toolUUID = s
}

func (this *nmapTooler) Parameters() string {
	params := `{
		"form": [
  {
    "type": "section",
    "htmlClass": "row",
    "items": [
      {
        "type": "section",
        "htmlClass": "col-xs-6",
        "items": [
          "scantype",
          {
            "key": "serviceversion",
            "type": "radiobuttons",
            "style": {
                "selected": "btn-success",
                "unselected": "btn-default"
            },
            "titleMap": [
              {
                "value": "false",
                "name": "Yes"
              },
              {
                "value": "true",
                "name": "No"
              }
            ]
          }
        ]
      },
      {
        "type": "section",
        "htmlClass": "col-xs-6",
        "items": [
          "timing",
          {
            "key": "skiphostdiscovery",
            "type": "radiobuttons",
            "style": {
                "selected": "btn-success",
                "unselected": "btn-default"
            },
            "titleMap": [
              {
                "value": "false",
                "name": "Yes"
              },
              {
                "value": "true",
                "name": "No"
              }
            ]
          }
        ]
      }
    ]
  },
  "ports",
  {
      "key": "portscustom",
      "condition": "model.ports == 'Custom'"
  },
  {
      "key": "targets",
      "type": "textarea"
  }
],
"schema": {
  "type": "object",
  "properties": {
    "scantype": {
        "title": "Scan type",
        "type": "string",
        "enum": [`
	var first = true
	for _, key := range getSortedKeys(scanTypes) {
		if !first {
			params += `,`
		}

		params += `"` + key + `"`
		first = false
	}
	params += `]
    },
    "serviceversion": {
      "title": "Enable service versioning?",
      "type": "string",
      "description": "Attempt to determine service (-sV)",
      "default": "false"
    },
    "skiphostdiscovery": {
      "title": "Skip discovering hosts?",
      "description": "Assume hosts are up (-PN)",
      "type": "string",
      "default": "true"
    },
    "timing": {
        "title": "Scan timing and performance",
        "type": "string",
        "enum": [`
	first = true
	for _, key := range timingOrder {
		if !first {
			params += `,`
		}

		params += `"` + key + `"`
		first = false
	}
	params += `],
        "default": "Normal (3)"
    },
    "ports": {
        "title": "Ports to scan",
        "type": "string",
        "default": "Most Common 1,000",
        "enum": [`
	first = true
	for _, key := range getSortedKeys(portSettings) {
		if !first {
			params += `,`
		}

		params += `"` + key + `"`
		first = false
	}
	params += `]
    },
    "portscustom": {
        "title": "Custom port listing",
        "type": "string",
        "description": "Enter a comma separated list of ports to scan",
        "pattern": "^\\d{1,5}(,\\d{1,5})*$",
        "validationMessage": "Only numbers and commas should be used, should use valid ports.  Ex: 22,25,80,443"
    },
    "targets": {
        "title": "Target Networks and Addresses",
        "type": "string",
        "description": "A listing of targets, one per line.",
        "pattern": "^[0-9-\\/\\.\\n]+$",
        "validationMessage": "Targets should be in CIDR (192.168.1.0/24), hyphenated (192.168.1-2.1-254), or address formats."
    }
  },
  "required": [
    "scantype",
    "timing",
    "ports",
    "targets"
  ]
}
}`
	return params
}

func (this *nmapTooler) Requirements() string {
	return common.RES_NET
}

func (this *nmapTooler) NewTask(job common.Job) (common.Tasker, error) {
	return newNmapTask(job)
}

func NewTooler() common.Tooler {
	return &nmapTooler{}
}

func getSortedKeys(src map[string]string) []string {
	keys := make([]string, len(src))

	i := 0
	for key, _ := range src {
		keys[i] = key
		i++
	}
	sort.Strings(keys)
	return keys
}
