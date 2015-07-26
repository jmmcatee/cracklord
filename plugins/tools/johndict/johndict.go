package johndict

import (
	"bytes"
	"encoding/csv"
	"errors"
	"os/exec"
	"sort"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/jmmcatee/cracklord/common"
	"github.com/vaughan0/go-ini"
)

/*
	Structure for configuration file
*/
type johndictConfig struct {
	BinPath         string
	JohnConfDir     string
	WorkingDir      string
	Arguments       string
	Dictionaries    map[string]string
	DictionaryOrder []string
	Rules           map[string]string
	RulesOrder      []string
	Formats         []string
}

/*
	Globals to be loaded into from the configuration file
*/
var config johndictConfig

// Setup function for the John Dictionary plugin
func Setup(path string) error {
	log.Debug("Setting up johndict tool")

	config = johndictConfig{Dictionaries: map[string]string{}, Rules: map[string]string{}}

	confFile, err := ini.LoadFile(path)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
			"file":  path,
		}).Error("Unable to load configuration file.")
		return err
	}

	basic := confFile.Section("Basic")
	if len(basic) == 0 {
		// Nothing retrieved, so return error
		return errors.New("No \"Basic\" configuration section.")
	}

	config.BinPath = basic["binPath"]
	config.WorkingDir = basic["workingdir"]
	config.Arguments = basic["arguments"]

	log.WithFields(log.Fields{
		"binpath":   config.BinPath,
		"WorkDir":   config.WorkingDir,
		"Arguments": config.Arguments,
	}).Debug("Basic configuration complete")

	// Run the executable to get the supported formats
	stdout, err := exec.Command(config.BinPath, "--list=formats").Output()
	if err != nil {
		// Something is wrong with our executable so log and fail
		log.WithField("error", err.Error()).Error("Could not pull format list.")
		return err
	}
	cleanedFormat := strings.Replace(strings.Replace(string(stdout), "\n", "", -1), " ", "", -1)
	formatBuf := bytes.NewBufferString(cleanedFormat)
	formatParsed := csv.NewReader(formatBuf)
	config.Formats, err = formatParsed.Read()
	if err != nil {
		log.WithField("error", err.Error()).Error("Could not parse the formats.")
		return err
	}
	sort.Strings(config.Formats)

	// Get the dictionary section
	dicts := confFile.Section("Dictionaries")
	if len(dicts) == 0 {
		// Nothing retrieved, so return error
		log.Debug("No 'dictionaries' configuration section.")
		return errors.New("No \"Dictionaries\" configuration section.")
	}
	for key, value := range dicts {
		log.WithFields(log.Fields{
			"name": key,
			"path": value,
		}).Debug("Added dictionary")
		config.Dictionaries[key] = value
	}

	// Get the rule section
	stdout, err = exec.Command(config.BinPath, "--list=rules").Output()
	if err != nil {
		// Something is wrong with our executable so log and fail
		log.WithField("error", err.Error()).Error("Could not pull format list.")
		return err
	}
	rules := strings.Split(string(stdout), "\n")
	for _, value := range rules {
		log.WithFields(log.Fields{
			"name": value,
			"path": value,
		}).Debug("Added rule file")
		config.Rules[value] = value
	}

	// Setup sorted order for consistency
	for key := range config.Dictionaries {
		config.DictionaryOrder = append(config.DictionaryOrder, key)
	}
	sort.Strings(config.DictionaryOrder)

	for key := range config.Rules {
		config.RulesOrder = append(config.RulesOrder, key)
	}
	sort.Strings(config.RulesOrder)

	log.Info("John Dictionary Attack tool successfully setup")
	return nil
}

/*
	Struct to hold the "tooler" information used by the resource server to setup
	the plugin and maintain it's state.  This is the information that allows the
	queue server to start jobs on the "tasker" portion.
	guesses: 0  time: 0:00:00:12 15.00% (ETA: Thu Jul  2 13:48:54 2015)  c/s: 20469
*/
type johndictTooler struct {
	toolUUID string
}

/*
	Return the name of the plugin.  This will be presented to users via the
	API when they select a tool
*/
func (h *johndictTooler) Name() string {
	return "John the Ripper - Dictionary Attack"
}

/*
	Return a string of the type of plugin.  This can be used via the API
	for categorization of tools.
*/
func (h *johndictTooler) Type() string {
	return "Dictionary"
}

/*
	Return the version of the tool.  In the event multiple versions of the same
	tool exist, all will be presented to the API.  Tools with the same name
	and version will only be presented once.  Jobs can only be started on one
	tool (name + version) at a time.
*/
func (h *johndictTooler) Version() string {
	return "1.7.9"
}

/*
	Return the UUID of this tool.  Note, if the same tool is running on multiple
	resources they may have different UUIDs, this is expected behavior, which is
	why version and name are used to determine duplicates of tools
*/
func (h *johndictTooler) UUID() string {
	return h.toolUUID
}

/*
	Set the UUID of the tool as necessary
*/
func (h *johndictTooler) SetUUID(s string) {
	h.toolUUID = s
}

/*
	Return information about what data is required to start a job utilizing this
	tool.  The first UI for CrackLord was written using AngularJS, so the schema
	for this information used a common form schema.

	For information regarding the schema, see schemaform.io.  To test the output
	of a schema, you can visit: http://schemaform.io/examples/bootstrap-example.html

	The form is split into two JSON strings, one array and one object. The
	first is the form array, which contains a listing of the fields that will be
	in the form.  There are some pre-defined keywords (see schemaform.io docs),
	otherwise each field takes several properties for that array element.

	Second is the schema object, whcih defines the details and validation
	requirements for each of the items named in the form array.  The API and
	default web GUI will process this information and present the form to the
	user, the queue and resource servers simply see these as strings to be
	passed, allowing a great deal of flexibility in the form.
*/
func (h *johndictTooler) Parameters() string {
	params := `{
		"form": [
			"algorithm",
		  	"dictionaries",
		  	"rules",
		  	{
		    	"key": "hashes",
		    	"type": "textarea",
		    	"placeholder": "Add in John required format"
		  	}
		],
		"schema": {
			"type": "object",
			  "properties": {
			    "name": {
			      "title": "Name",
			      "type": "string"
			    },
			    "algorithm": {
			      "title": "Select hash type to attack",
			      "type": "string",
		     	 "enum": [ `
	var first = true
	for _, fstring := range config.Formats {
		if !first {
			params += `,`
		}

		params += `"` + fstring + `"`

		first = false
	}

	params += `
		]
	   },
	    "dictionaries": {
	      "title": "Select dictionary to use",
	      "type": "string",
	      "enum": [ `

	first = true
	for _, key := range config.DictionaryOrder {
		if !first {
			params += `,`
		}

		params += `"` + key + `"`

		first = false
	}

	params += `
      ]
    },
    "rules": {
      "title": "Select rule file to use",
      "type": "string",
      "enum": [ `

	first = true
	for _, key := range config.RulesOrder {
		if !first {
			params += `,`
		}

		params += `"` + key + `"`

		first = false
	}

	params += ` ]
	    },
	    "customdictadd": {
	      "title": "Custom Dictionary Additions",
	      "type": "string"
	    },
	    "hashes": {
	      "title": "Hashes",
	      "type": "string"
	    }
	  },
	  "required": [
	    "name",
	    "algorithm",
	    "dictionaries",
	    "hashes"
	  ]
	} } `

	return params
}

/*
	Return the type of resource that will be used by this tool.  Typically this
	will be either GPU or CPU; however, additional types can be configured
	in common.go as necessary.  The queue will manage jobs by sending them to
	resources with the necessary resources available.
*/
func (h *johndictTooler) Requirements() string {
	return common.RES_CPU
}

/*
	Start a new job by using the tasker for this tool
*/
func (h *johndictTooler) NewTask(job common.Job) (common.Tasker, error) {
	return newJohnDictTask(job)
}

// NewTooler function for creating a common.Tooler for the John Dictionary Plugin
func NewTooler() common.Tooler {
	return &johndictTooler{}
}
