package hashcat

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/jmmcatee/cracklord/common"
	"github.com/jmmcatee/goschemaform"
	"github.com/vaughan0/go-ini"
	"sort"
)

type hcConfig struct {
	BinPath       string
	WorkDir       string
	Arguments     string
	Dictionaries  dictionaries
	Rules         rules
	CharacterSets charactersets
}

var config = hcConfig{
	BinPath:   "",
	WorkDir:   "",
	Arguments: "",
}

/*
	Read the hascatdict init file to setup hashcat
*/
func Setup(path string) error {
	log.Debug("Setting up hashcat tool")
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
		config.Dictionaries = append(config.Dictionaries, dictionary{Name: key, Path: value})
	}

	// Get the rule section
	rules := confFile.Section("Rules")
	if len(rules) == 0 {
		// Nothing retrieved, so return error
		log.Debug("No 'rules' configuration section.")
		return errors.New("No \"Rules\" configuration section.")
	}
	for key, value := range rules {
		log.WithFields(log.Fields{
			"name": key,
			"path": value,
		}).Debug("Added rule")
		config.Rules = append(config.Rules, rule{Name: key, Path: value})
	}

	// Store the character sets configured for brute forcing in the config file
	charset := confFile.Section("BruteCharset")
	if len(charset) == 0 {
		// Nothing retrieved, so return error
		log.Debug("No 'charset' configuration section.")
		return errors.New("No \"charset\" configuration section.")
	}
	for key, value := range charset {
		log.WithFields(log.Fields{
			"name": key,
			"path": value,
		}).Debug("Added charset to hashcat")
		config.CharacterSets = append(config.CharacterSets, characterset{Name: key, Mask: value})
	}

	log.Info("Hashcat tool successfully setup")

	return nil
}

type hashcatTooler struct {
	toolUUID string
}

func (h *hashcatTooler) Name() string {
	return "oclHashcat"
}

func (h *hashcatTooler) Type() string {
	return "Password Cracking"
}

func (h *hashcatTooler) Version() string {
	return "2.01"
}

func (h *hashcatTooler) UUID() string {
	return h.toolUUID
}

func (h *hashcatTooler) SetUUID(s string) {
	h.toolUUID = s
}

func (h *hashcatTooler) Parameters() string {
	hashcatForm := goschemaform.NewSchemaForm()

	// Setup the dropdown for the hashing algorithm to use
	algoInput := goschemaform.NewDropDownInput("algorithm")
	algoInput.SetTitle("Select hash type to attack")
	algoInput.IsRequired(true)
	sort.Sort(hashAlgorithms(algorithms))
	for i := range algorithms {
		option := goschemaform.NewDropDownInputOption(algorithms[i].Number)
		option.SetGroup(algorithms[i].Group)
		option.SetName(algorithms[i].Name)
		algoInput.AddOption(option)
	}
	// Add the dropdown to the form at the top
	hashcatForm.AddElement(algoInput)

	// Build the fieldset for tabs based on attack type (Dictionary vs Bruteforce)
	attackTypeFieldset := goschemaform.NewTabFieldset()
	attackTypeFieldset.SetTitle("Attack Type")

	// Build the dictionary attack tab
	dictionaryAttackTab := goschemaform.NewTab()
	dictionaryAttackTab.SetTitle("Dictionary")
	// Setup the dropdown for choosing a dictionary to use
	dictionaryDropDown := goschemaform.NewDropDownInput("dict_dictionaries")
	dictionaryDropDown.SetTitle("Select dictionary to use")
	sort.Sort(config.Dictionaries)
	for i := range config.Dictionaries {
		option := goschemaform.NewDropDownInputOption(config.Dictionaries[i].Name)
		dictionaryDropDown.AddOption(option)
	}
	// Add the dictionary drop down to the tab
	dictionaryAttackTab.AddElement(dictionaryDropDown)
	// Build the rules dropdown
	ruleDropDown := goschemaform.NewDropDownInput("dict_rules")
	ruleDropDown.SetTitle("Select rule file to use")
	sort.Sort(config.Rules)
	for i := range config.Rules {
		option := goschemaform.NewDropDownInputOption(config.Rules[i].Name)
		ruleDropDown.AddOption(option)
	}
	// Add the rules drop down to the tab
	dictionaryAttackTab.AddElement(ruleDropDown)
	// Add the tab to the Attack Type fieldset
	attackTypeFieldset.AddTab(dictionaryAttackTab)

	// Buld the bruteforce attack tab
	bruteForceTab := goschemaform.NewTab()
	bruteForceTab.SetTitle("Brute Force")
	// Setup the input for getting the length of the bruteforce
	bfLength := goschemaform.NewNumberInput("brute_length")
	bfLength.SetTitle("Select the length of the charset")
	bfLength.SetMin(0)
	// Add Length input to the tab
	bruteForceTab.AddElement(bfLength)
	// Add whether to increment from 0 - length
	bfIncrementCheckBox := goschemaform.NewCheckBoxInput("brute_increment")
	bfIncrementCheckBox.SetTitle("Check for incremental mode")
	// Add the checkbox to the tab
	bruteForceTab.AddElement(bfIncrementCheckBox)
	// Setup the dropdown for choosing a character set
	bfCharSetDropDown := goschemaform.NewDropDownInput("brute_charset")
	bfCharSetDropDown.SetTitle("Select character set")
	sort.Sort(config.CharacterSets)
	for i := range config.CharacterSets {
		option := goschemaform.NewDropDownInputOption(config.CharacterSets[i].Name)
		bfCharSetDropDown.AddOption(option)
	}
	// Add the dropdown to the tab
	bruteForceTab.AddElement(bfCharSetDropDown)
	// Add the tab to the Attack Type fieldset
	attackTypeFieldset.AddTab(bruteForceTab)

	// Add the tab fieldset to the form
	hashcatForm.AddElement(attackTypeFieldset)

	// Build the hashes multiline input
	hashesMultiline := goschemaform.NewTextInput("hashes")
	hashesMultiline.SetTitle("Hashes")
	hashesMultiline.SetPlaceHolder("Add in Hashcat required format")
	hashesMultiline.SetMultiline(true)
	hashesMultiline.IsRequired(true)
	// Add the multiline to the form
	hashcatForm.AddElement(hashesMultiline)

	return hashcatForm.SchemaForm()
}

func (h *hashcatTooler) Requirements() string {
	return common.RES_GPU
}

func (h *hashcatTooler) NewTask(job common.Job) (common.Tasker, error) {
	return newHashcatTask(job)
}

func NewTooler() common.Tooler {
	return &hashcatTooler{}
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
