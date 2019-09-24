package hashcat5

import (
	"encoding/base64"
	"errors"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/jmmcatee/cracklord/common"
	"github.com/jmmcatee/goschemaform"
)

const (
	USER_HASHES_FILENAME      = "user-input-hashes.txt"
	HASHCAT_POT_SHOW_FILENAME = "hashcat-pot-show.txt"
	HASHCAT_LEFT_FILENAME     = "hashcat-left.txt"
	HASH_OUTPUT_FILENAME      = "output-hashes.txt"
)

type hashcat5Tooler struct {
	toolUUID string
	version  string
}

func (h *hashcat5Tooler) Name() string {
	return "Hashcat"
}

func (h *hashcat5Tooler) Type() string {
	return "Password Cracking"
}

func (h *hashcat5Tooler) Version() string {
	return h.version
}

func (h *hashcat5Tooler) UUID() string {
	return h.toolUUID
}

func (h *hashcat5Tooler) SetUUID(s string) {
	h.toolUUID = s
}

func (h *hashcat5Tooler) Requirements() string {
	return common.RES_GPU
}

// NewTooler returns a hashcat3 impementation of the common.Tooler
func NewTooler() common.Tooler {
	// Get the version from hashcat
	version, err := exec.Command(config.BinPath, "--version").Output()
	if err != nil {
		// This should not happen as the executable has already run once during the
		// Setup command. It is a possible error, but not sure how to hanlde it without
		// Update the interface, which we can do later.
		log.WithField("error", err.Error()).Error("Could not pull hashcat 5.x version")
	}

	tooler := &hashcat5Tooler{}
	tooler.version = string(version)

	return tooler
}

func (h *hashcat5Tooler) Parameters() string {
	hashcatForm := goschemaform.NewSchemaForm()

	// Setup the dropdown for the hashing hashmode to use
	hashModeInput := goschemaform.NewDropDownInput("hashmode")
	hashModeInput.SetTitle("Select hash type to attack")
	hashModeInput.IsRequired(true)

	for i := range config.HashModes {
		option := goschemaform.NewDropDownInputOption(config.HashModes[i].Number)
		option.SetGroup(config.HashModes[i].Category)
		option.SetName(config.HashModes[i].Name)
		hashModeInput.AddOption(option)
	}
	// Add the dropdown to the form at the top
	hashcatForm.AddElement(hashModeInput)

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

	// Setup checkbox to determine if we are prepending rules to the dictionary
	dictionaryPrependCheckbox := goschemaform.NewCheckBoxInput("dict_use_custom_prepend")
	dictionaryPrependCheckbox.SetTitle("Prepend custom words to the selected dictionary")
	// Add checkbox to to the tab
	dictionaryAttackTab.AddElement(dictionaryPrependCheckbox)

	// Setup conditional multiline if we want to prepend words to the dictionary
	dictionaryPrependMultiline := goschemaform.NewTextInput("dict_custom_prepend")
	dictionaryPrependMultiline.SetTitle("Custom Words to Prepend")
	dictionaryPrependMultiline.SetPlaceHolder("One word per line")
	dictionaryPrependMultiline.SetMultiline(true)
	dictionaryPrependMultiline.SetCondition("dict_use_custom_prepend", false)
	// Add dictionary prepend multiline to the tab
	dictionaryAttackTab.AddElement(dictionaryPrependMultiline)

	// Add a checkbox to determine if we want to also generate random rules
	ruleGenRandomCheckbox := goschemaform.NewCheckBoxInput("dict_rules_use_random")
	ruleGenRandomCheckbox.SetTitle("Generate random rules")
	// Add checkbox to the tab
	dictionaryAttackTab.AddElement(ruleGenRandomCheckbox)

	// Build a number input for the maximum number of random rules
	ruleGenRandomMax := goschemaform.NewNumberInput("dict_rules_random_max")
	ruleGenRandomMax.SetTitle("Maximum random rules to generate")
	ruleGenRandomMax.SetMin(1)
	ruleGenRandomMax.SetCondition("dict_rules_use_random", false)
	// Add input to tab
	dictionaryAttackTab.AddElement(ruleGenRandomMax)

	// Build a checkbox to determine if we are going to use an existing rules file or upload
	// a custom one for this job.
	ruleCustomCheckbox := goschemaform.NewCheckBoxInput("dict_rules_use_custom")
	ruleCustomCheckbox.SetTitle("Upload a custom rule file")
	ruleCustomCheckbox.SetCondition("dict_rules_use_random", true)
	// Add the checkbox to the form
	dictionaryAttackTab.AddElement(ruleCustomCheckbox)

	// Build the rules dropdown
	ruleDropDown := goschemaform.NewDropDownInput("dict_rules")
	ruleDropDown.SetTitle("Select rule file to use")
	ruleDropDown.SetCondition("dict_rules_use_custom && !model.dict_rules_use_random", true)

	for i := range config.RuleFiles {
		option := goschemaform.NewDropDownInputOption(config.RuleFiles[i].Name)
		ruleDropDown.AddOption(option)
	}
	// Add the rules drop down to the tab
	dictionaryAttackTab.AddElement(ruleDropDown)

	// Build a custom rule upload control
	ruleCustomUpload := goschemaform.NewFileInput("dict_rules_custom_file")
	ruleCustomUpload.SetTitle("Custom Rule File")
	ruleCustomUpload.SetPlaceHolder("Click here or drop file to upload")
	ruleCustomUpload.SetCondition("dict_rules_use_custom", false)
	// Add custom upload to the tab
	dictionaryAttackTab.AddElement(ruleCustomUpload)

	// Add the tab to the Attack Type fieldset
	attackTypeFieldset.AddTab(dictionaryAttackTab)

	// Buld the bruteforce attack tab
	bruteForceTab := goschemaform.NewTab()
	bruteForceTab.SetTitle("Brute Force")

	// Build a checkbox to select custom or predefined character sets
	bfCustomCharsets := goschemaform.NewCheckBoxInput("brute_use_custom_chars")
	bfCustomCharsets.SetTitle("Use a custom character set instead")
	// Add to the tab
	bruteForceTab.AddElement(bfCustomCharsets)

	// Build inputs for a custom character set
	// Custom Mask
	bfCustomMask := goschemaform.NewTextInput("brute_custom_mask")
	bfCustomMask.SetTitle("Custom Brute Force Mask")
	bfCustomMask.SetPlaceHolder("Mask to brute force...")
	bfCustomMask.SetMultiline(false)
	bfCustomMask.SetCondition("brute_use_custom_chars", false)
	// Add to the tab
	bruteForceTab.AddElement(bfCustomMask)
	// Custom Character Set 1
	bfCustomChar1 := goschemaform.NewTextInput("brute_custom_charset1")
	bfCustomChar1.SetTitle("Custom Character Set 1")
	bfCustomChar1.SetPlaceHolder("Custom characters...")
	bfCustomChar1.SetMultiline(false)
	bfCustomChar1.SetCondition("brute_use_custom_chars", false)
	// Add to the tab
	bruteForceTab.AddElement(bfCustomChar1)
	// Custom Character Set 2
	bfCustomChar2 := goschemaform.NewTextInput("brute_custom_charset2")
	bfCustomChar2.SetTitle("Custom Character Set 2")
	bfCustomChar2.SetPlaceHolder("Custom characters...")
	bfCustomChar2.SetMultiline(false)
	bfCustomChar2.SetCondition("brute_use_custom_chars", false)
	// Add to the tab
	bruteForceTab.AddElement(bfCustomChar2)
	// Custom Character Set 3
	bfCustomChar3 := goschemaform.NewTextInput("brute_custom_charset3")
	bfCustomChar3.SetTitle("Custom Character Set 3")
	bfCustomChar3.SetPlaceHolder("Custom characters...")
	bfCustomChar3.SetMultiline(false)
	bfCustomChar3.SetCondition("brute_use_custom_chars", false)
	// Add to the tab
	bruteForceTab.AddElement(bfCustomChar3)
	// Custom Character Set 4
	bfCustomChar4 := goschemaform.NewTextInput("brute_custom_charset4")
	bfCustomChar4.SetTitle("Custom Character Set 4")
	bfCustomChar4.SetPlaceHolder("Custom characters...")
	bfCustomChar4.SetMultiline(false)
	bfCustomChar4.SetCondition("brute_use_custom_chars", false)
	// Add to the tab
	bruteForceTab.AddElement(bfCustomChar4)

	// Setup the dropdown for choosing a character set
	bfCharSetDropDown := goschemaform.NewDropDownInput("brute_predefined_charset")
	bfCharSetDropDown.SetTitle("Select character set (?1=?l?d, ?2=?u?l?d, ?3=?d?s, ?4=?l?d?s)")
	bfCharSetDropDown.SetCondition("brute_use_custom_chars", true)

	for i := range config.Charsets {
		option := goschemaform.NewDropDownInputOption(config.Charsets[i].Name)
		bfCharSetDropDown.AddOption(option)
	}
	// Add the dropdown to the tab
	bruteForceTab.AddElement(bfCharSetDropDown)

	// Add whether to increment from minLenght - maxLength
	bfIncrementCheckBox := goschemaform.NewCheckBoxInput("brute_increment")
	bfIncrementCheckBox.SetTitle("Enabled incremental mode")
	// Add the checkbox to the tab
	bruteForceTab.AddElement(bfIncrementCheckBox)

	// Setup the starting value of the incremental mode
	bfMinLength := goschemaform.NewNumberInput("brute_min_length")
	bfMinLength.SetTitle("Select the starting length of the charset")
	bfMinLength.SetMin(1)
	bfMinLength.SetCondition("brute_increment", false)
	// Add Length input to the tab
	bruteForceTab.AddElement(bfMinLength)

	// Setup the input for getting the length of the bruteforce
	bfMaxLength := goschemaform.NewNumberInput("brute_max_length")
	bfMaxLength.SetTitle("Select the maximum length of the charset")
	bfMaxLength.SetMin(1)
	bfMaxLength.SetCondition("brute_increment", false)
	// Add Length input to the tab
	bruteForceTab.AddElement(bfMaxLength)

	// Add the tab to the Attack Type fieldset
	attackTypeFieldset.AddTab(bruteForceTab)

	// Add the tab fieldset to the form
	hashcatForm.AddElement(attackTypeFieldset)

	// Build the fieldset for the Hash input options
	hashFieldset := goschemaform.NewTabFieldset()
	hashFieldset.SetTitle("Hash Input")

	// Only need one tab
	hashTab := goschemaform.NewTab()
	hashTab.SetTitle("Hashes")

	// Build a checkbox to determine how we will upload hashes
	hashFileUploadCheckbox := goschemaform.NewCheckBoxInput("hashes_use_upload")
	hashFileUploadCheckbox.SetTitle("Use a file to upload hashes")
	// Add to the tab
	hashTab.AddElement(hashFileUploadCheckbox)

	// Build the hashes multiline input
	hashesMultiline := goschemaform.NewTextInput("hashes_multiline")
	hashesMultiline.SetTitle("Hashes")
	hashesMultiline.SetPlaceHolder("Add in Hashcat required format")
	hashesMultiline.SetMultiline(true)
	hashesMultiline.IsRequired(true)
	hashesMultiline.SetCondition("hashes_use_upload", true)
	// Add to the tab
	hashTab.AddElement(hashesMultiline)

	// Build the hash file upload
	hashesFileUpload := goschemaform.NewFileInput("hashes_file_upload")
	hashesFileUpload.SetTitle("Hashes File")
	hashesFileUpload.SetPlaceHolder("Click here or drop file to upload")
	hashesFileUpload.SetCondition("hashes_use_upload", false)
	// Add to the tab
	hashTab.AddElement(hashesFileUpload)

	// Add the tab to the fieldset and the fieldset to the form
	hashFieldset.AddTab(hashTab)
	hashcatForm.AddElement(hashFieldset)

	// Build a checkbox to show or hide advanced options
	advancedOptionsCheckbox := goschemaform.NewCheckBoxInput("use_adv_options")
	advancedOptionsCheckbox.SetTitle("Show advanced options")

	// Add checkbox to the form
	hashcatForm.AddElement(advancedOptionsCheckbox)

	// Build fieldset and tab for the advanced options
	advancedOptionsFieldset := goschemaform.NewTabFieldset()
	advancedOptionsFieldset.SetTitle("Advanced Options")
	advancedOptionsFieldset.SetCondition("use_adv_options", false)

	// Build the tabs
	// Loopback
	advOptTabLoopback := goschemaform.NewTab()
	advOptTabLoopback.SetTitle("Loopback Input")
	advOptLookbackCheckbox := goschemaform.NewCheckBoxInput("adv_options_loopback")
	advOptLookbackCheckbox.SetTitle("Enable loopback flag")
	advOptTabLoopback.AddElement(advOptLookbackCheckbox)
	// Add tab
	advancedOptionsFieldset.AddTab(advOptTabLoopback)
	// Markov
	advOptTabMarkov := goschemaform.NewTab()
	advOptTabMarkov.SetTitle("Markov Options")
	advOptMarkovNumber := goschemaform.NewNumberInput("adv_options_markov")
	advOptMarkovNumber.SetTitle("Markov Threshold")
	advOptMarkovNumber.SetMin(0)
	advOptTabMarkov.AddElement(advOptMarkovNumber)
	// Add tab
	advancedOptionsFieldset.AddTab(advOptTabMarkov)
	// Timeout
	advOptTabTimeout := goschemaform.NewTab()
	advOptTabTimeout.SetTitle("Timeout Options")
	advOptTimeoutNumber := goschemaform.NewNumberInput("adv_options_timeout")
	advOptTimeoutNumber.SetTitle("Job Timeout (in seconds)")
	advOptTimeoutNumber.SetMin(120)
	advOptTabTimeout.AddElement(advOptTimeoutNumber)
	// Add tab
	advancedOptionsFieldset.AddTab(advOptTabTimeout)

	// Add fieldset to the form
	hashcatForm.AddElement(advancedOptionsFieldset)

	return hashcatForm.SchemaForm()
}

func (h *hashcat5Tooler) NewTask(job common.Job) (common.Tasker, error) {
	t := Tasker{}

	t.job = job

	var err error
	t.wd, err = createWorkingDir(config.WorkingDir, t.job.UUID)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Build the arguements for hashcat
	args := []string{} // all of it put together
	opts := []string{} // [options]
	var argHash string // hash|hashfile|hccapfile
	var argDmD string  // [dictionary|mask|directory]

	logParam := map[string]string{}

	for k, v := range t.job.Parameters {
		if k != "hashes_multiline" && k != "hashes_file_upload" {
			logParam[k] = v
		}
	}

	log.WithField("params", logParam).Debug("Create Hashcat Job Parameters.")

	// Get the hash type and add an argument
	htype, ok := t.job.Parameters["hashmode"]
	if !ok {
		log.WithFields(log.Fields{
			"hashmode": htype,
			"err":      ok,
		}).Error("Could not find the hashmode provided")
		return nil, errors.New("Could not find the hashmode provided.")
	}
	opts = append(opts, "--hash-type="+htype)
	t.hashMode = htype

	var modeSet bool
	/////////////////////////////////////////////////////////////////////////////////////////
	// Check for Dictionary Crack mode
	if dictDictionary, dictionaryOk := t.job.Parameters["dict_dictionaries"]; dictionaryOk {
		log.Debug("Dictionary attack selected.")
		opts = append(opts, "--attack-mode", "0")
		modeSet = true

		// Check the dictionary is one we have
		dictIndex := sort.Search(len(config.Dictionaries), func(i int) bool { return config.Dictionaries[i].Name >= dictDictionary })
		if dictIndex == len(config.Dictionaries) {
			// We did not find the dictionary so return an error
			log.WithField("dictionary", dictDictionary).Error("Dictionary provided does not exist.")
			return nil, errors.New("Dictionary provided does not exist.")
		}
		log.WithField("Dictionary", config.Dictionaries[dictIndex].Path).Debug("Dictionary selected.")

		// Check for custom dictionary prepend
		var dictPrependBool bool
		if dictPrependString, prependOk := t.job.Parameters["dict_use_custom_prepend"]; prependOk {
			dictPrependBool, err = strconv.ParseBool(dictPrependString)
			if err != nil {
				log.WithFields(log.Fields{
					"error":      err,
					"boolString": dictPrependString,
				}).Error("Error parsing a bool")
				return nil, err
			}
		}

		dictPrependCustom, dictPrependCustomOk := t.job.Parameters["dict_custom_prepend"]
		if dictPrependBool && dictPrependCustomOk {
			log.Debug("Custom Prepend value to dictionary attack selected.")
			// We need to get the dictionary file to copy into the working directory
			customDictPath := filepath.Join(t.wd, "custom-prepend-dict.txt")

			err := common.CopyPrepend(customDictPath, config.Dictionaries[dictIndex].Path, dictPrependCustom)
			if err != nil {
				// Something went wrong in the file copy
				log.WithField("Copy Error", err).Error("Error copying dictionary")
				return nil, err
			}

			// We now have our new dictionary file so append this to the arguments
			argDmD = customDictPath
			log.WithField("Custom dictionary path", customDictPath)
		} else {
			// We are not using a custom dictionary so use the one provided that we know is valid
			argDmD = config.Dictionaries[dictIndex].Path
		}

		// Check if we are using any rule files or generating them randomly
		useRandomRuleString, useRandomRulesOk := t.job.Parameters["dict_rules_use_random"]
		ruleRandomMaxString, ruleRandomMaxOk := t.job.Parameters["dict_rules_random_max"]
		useRandomRuleBool, err := strconv.ParseBool(useRandomRuleString)
		if err != nil && useRandomRulesOk {
			log.WithFields(log.Fields{
				"error":      err,
				"boolString": useRandomRuleString,
			}).Error("Error parsing a bool")
			return nil, err
		}

		if useRandomRuleBool && ruleRandomMaxOk {
			// We have been told to use a random rule and have a random rule value

			// Parse the random value
			ruleRandomMax, err := strconv.Atoi(ruleRandomMaxString)
			if err != nil {
				log.WithFields(log.Fields{
					"error":     err,
					"intString": ruleRandomMaxString,
				}).Error("Error parsing int")
				return nil, err
			}

			if ruleRandomMax <= 0 {
				log.Error("The value given for the number of random rules to generate was not more than 0.")
				return nil, errors.New("The value given for the number of random rules to generate was not more than 0.")
			}

			// Append the value
			opts = append(opts, "--generate-rules="+ruleRandomMaxString)
		} else {
			// We are not using random rules so we are going to provide a rule file, upload a rule file, or not use rules at all
			log.Debug("No using randomly generated rules")

			var ruleUseCustomBool bool
			if ruleUseCustomString, useRuleCustomOk := t.job.Parameters["dict_rules_use_custom"]; useRuleCustomOk {
				if ruleUseCustomBool, err = strconv.ParseBool(ruleUseCustomString); err != nil {
					log.WithFields(log.Fields{
						"error":      err,
						"boolString": ruleUseCustomString,
					}).Error("Error parsing a bool")
					return nil, err
				}
			}

			ruleFile, ruleFileOk := t.job.Parameters["dict_rules"]
			_, ruleCustomFileOk := t.job.Parameters["dict_rules_custom_file"] // Don't copy the file in memory yet if we have it (might be big)
			if ruleFileOk {
				// We are going to use a preconfigured rule file
				log.Debug("Using a preconfigured rule file.")

				// Check that we were given a valid preconfigured rule
				ruleIndex := sort.Search(len(config.RuleFiles), func(i int) bool { return config.RuleFiles[i].Name >= ruleFile })
				if ruleIndex == len(config.RuleFiles) {
					// We did not find the rule file provided
					log.WithField("rule file", ruleFile).Error("Rule file selected does not exit.")
					return nil, errors.New("Rule file provided does not exist.")
				}

				// Add the fule file argument
				opts = append(opts, "--rules-file", config.RuleFiles[ruleIndex].Path)
				log.WithField("rules", config.RuleFiles[ruleIndex].Path).Debug("Rule file selected")

			} else if ruleUseCustomBool && ruleCustomFileOk {
				// We are going to use a custom uploaded rule file
				log.Debug("Using custom uploaded rule file")

				// Get the rule file provided, parse it and write it to disk
				fileParts := strings.Split((t.job.Parameters["dict_rules_custom_file"]), ";")
				if len(fileParts) != 3 {
					log.Error("Error parsing the uploaded file.")
					return nil, errors.New("Error parsing the uploaded file.")
				}
				// [0] - file:[filename of uploaded file]
				// [1] - data:[data type (text/plain)]
				// [2] - base64,[data]

				// Decode the file
				customRuleFilePath := filepath.Join(t.wd, "custom-uploaded-rules.txt")
				customRuleFileBytes, err := base64.StdEncoding.DecodeString(fileParts[2][7:])
				if err != nil {
					log.WithField("error", err).Error("Error parsing hex value of uploaded rule file.")
					return nil, err
				}

				// write the file to disk
				err = ioutil.WriteFile(customRuleFilePath, customRuleFileBytes, 0666)
				if err != nil {
					log.WithField("error", err).Error("Error writing the uploaded rule file to disk.")
					return nil, err
				}

				// Append the file to the args
				opts = append(opts, "--rules-file", customRuleFilePath)
				log.WithField("rules", customRuleFilePath).Debug("Rule file uploaded")
			}
		}
	}

	/////////////////////////////////////////////////////////////////////////////////////////
	// Check for Brute Force Crack mode
	var bruUseCustomMaskBool bool
	if bruUseCustomMaskString, useCustomOk := t.job.Parameters["brute_use_custom_chars"]; useCustomOk {
		if bruUseCustomMaskBool, err = strconv.ParseBool(bruUseCustomMaskString); err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"boolString": bruUseCustomMaskString,
			}).Error("Error parsing a bool")
			return nil, err
		}
	}

	bruCustomMask, custMaskOk := t.job.Parameters["brute_custom_mask"]
	bruPreDefMask, preDefMaskOk := t.job.Parameters["brute_predefined_charset"]
	if bruUseCustomMaskBool && custMaskOk {
		log.Debug("Use custome character sets.")
		opts = append(opts, "--attack-mode", "3")
		modeSet = true

		// We are provided a custom mask and maybe custom character sets.
		// Check custom character sets 1
		if bruCustomChar1, bruCustomChar1Ok := t.job.Parameters["brute_custom_charset1"]; bruCustomChar1Ok {
			if len(bruCustomChar1) > 0 {
				// We were given a custom character set and its length is non-zero so add it as a flag
				opts = append(opts, "--custom-charset1="+bruCustomChar1)
			}
		}

		// Check custom character sets 2
		if bruCustomChar2, bruCustomChar2Ok := t.job.Parameters["brute_custom_charset2"]; bruCustomChar2Ok {
			if len(bruCustomChar2) > 0 {
				// We were given a custom character set and its length is non-zero so add it as a flag
				opts = append(opts, "--custom-charset2="+bruCustomChar2)
			}
		}

		// Check custom character sets 3
		if bruCustomChar3, bruCustomChar3Ok := t.job.Parameters["brute_custom_charset3"]; bruCustomChar3Ok {
			if len(bruCustomChar3) > 0 {
				// We were given a custom character set and its length is non-zero so add it as a flag
				opts = append(opts, "--custom-charset3="+bruCustomChar3)
			}
		}

		// Check custom character sets 4
		if bruCustomChar4, bruCustomChar4Ok := t.job.Parameters["brute_custom_charset4"]; bruCustomChar4Ok {
			if len(bruCustomChar4) > 0 {
				// We were given a custom character set and its length is non-zero so add it as a flag
				opts = append(opts, "--custom-charset4="+bruCustomChar4)
			}
		}

		// We have now defined our custom character sets, if any, so check our provided custom mask
		if len(bruCustomMask) <= 0 {
			log.WithField("brute_custom_mask", bruCustomMask).Error("Length of brute_custom_mask was <= 0.")
			return nil, errors.New("Length of brute_custom_mask was <= 0.")
		}

		// Append the custom mask
		argDmD = bruCustomMask
	} else if preDefMaskOk {
		log.Debug("Do not use custom character sets")
		opts = append(opts, "--attack-mode", "3")
		modeSet = true

		// We selected a preconfigured mask so make sure it exists
		charSetIndex := sort.Search(len(config.Charsets), func(i int) bool { return config.Charsets[i].Name >= bruPreDefMask })
		if charSetIndex == len(config.Charsets) {
			// We did not find the dictionary so return an error
			log.WithField("characterset", bruPreDefMask).Error("Character Set provided does not exist.")
			return nil, errors.New("Character Set provided does not exist.")
		}

		// Add the custom character sets used by the configured pre definied character masks
		opts = append(opts, "--custom-charset1="+CharSetPreDefCustom1)
		opts = append(opts, "--custom-charset2="+CharSetPreDefCustom2)
		opts = append(opts, "--custom-charset3="+CharSetPreDefCustom3)
		opts = append(opts, "--custom-charset4="+CharSetPreDefCustom4)

		// The mask provided is good so append to arguments
		argDmD = config.Charsets[charSetIndex].Mask
	}

	if (bruUseCustomMaskBool && custMaskOk) || preDefMaskOk {
		log.Debug("We are doing a bruteforce crack and need to check the incremental mode")

		var incModeBool bool
		if incModeString, incModeOk := t.job.Parameters["brute_increment"]; incModeOk {
			if incModeBool, err = strconv.ParseBool(incModeString); err != nil {
				log.WithFields(log.Fields{
					"error":      err,
					"boolString": incModeString,
				}).Error("Error parsing a bool")
			}

			// Enable increment mode
			opts = append(opts, "--increment")

			if incModeBool {
				// Check the start and maxium integers
				if incMinString, incMinOk := t.job.Parameters["brute_min_length"]; incMinOk {
					// parse the int and validate
					var incMinInt int
					if incMinInt, err = strconv.Atoi(incMinString); err != nil {
						log.WithField("error", err).Error("Error parsing brute_min_length into int.")
						return nil, err
					}

					if incMinInt <= 0 {
						log.WithField("brute_min_length", incMinString).Error("The brute_min_length value is not at least 1.")
						return nil, errors.New("The brute_min_length value is not at least 1.")
					}

					// Append the minimum value
					opts = append(opts, "--increment-min="+incMinString)
				}

				if incMaxString, incMaxOk := t.job.Parameters["brute_max_length"]; incMaxOk {
					// parse the int and validate
					var incMaxInt int
					if incMaxInt, err = strconv.Atoi(incMaxString); err != nil {
						log.WithField("error", err).Error("Error parsing brute_max_length into int.")
						return nil, err
					}

					if incMaxInt <= 0 {
						log.WithField("brute_max_length", incMaxString).Error("The brute_max_length value is not at least 1.")
						return nil, errors.New("The brute_max_length value is not at least 1.")
					}

					// Append the minimum value
					opts = append(opts, "--increment-max="+incMaxString)
				}
			}
		}
	}

	// Check that we set a mode, if not something is wrong so fail
	if !modeSet {
		log.Error("No attack mode was set.")
	}

	// Start parsing the hash input
	var hashUseUploadBool bool
	if hashUseUploadString, hashUseUploadOk := t.job.Parameters["hashes_use_upload"]; hashUseUploadOk {
		if hashUseUploadBool, err = strconv.ParseBool(hashUseUploadString); err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"boolString": hashUseUploadString,
			}).Error("Error parsing a bool")
			return nil, err
		}
	}

	var hashBytes []byte
	hashFilePath := filepath.Join(t.wd, USER_HASHES_FILENAME)
	if hashUseUploadBool {
		// Use an uploaded hashfile
		if _, hashUploadOk := t.job.Parameters["hashes_file_upload"]; hashUploadOk {
			// Get the hash file provided, parse it and write it to disk
			hashBytes, err = decodeBase64Upload(t.job.Parameters["hashes_file_upload"])
			if err != nil {
				// We should have already written the error to the log so just return
				return nil, err
			}
		} else {
			log.Error("No hash file was uploaded, even though we checked the box.")
			return nil, errors.New("No hash file was uploaded, even though we checked the box.")
		}
	} else {
		// Use the provided text input for the hashfile
		if _, hashMultilineOk := t.job.Parameters["hashes_multiline"]; hashMultilineOk {
			if len(t.job.Parameters["hashes_multiline"]) <= 0 {
				log.Error("The hash multiline was provided with a length of 0.")
				return nil, errors.New("The hash multiline was provided with a length of 0.")
			}

			hashBytes = []byte(t.job.Parameters["hashes_multiline"])
		} else {
			log.Error("Hashes were not provided in the multiline input and no file was uploaded.")
			return nil, errors.New("Hashes were not provided in the multiline input and no file was uploaded.")
		}
	}

	// Save hashes to a file for us to process later
	err = ioutil.WriteFile(filepath.Join(t.wd, USER_HASHES_FILENAME), hashBytes, 0660)
	if err != nil {
		return nil, err
	}

	// Add the hash to the arguments variable
	argHash = hashFilePath

	// Parse any advanced options provided
	var advancedOptionsBool bool
	if advancedOptionsString, advancedOptionsOk := t.job.Parameters["use_adv_options"]; advancedOptionsOk {
		if advancedOptionsBool, err = strconv.ParseBool(advancedOptionsString); err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"boolString": advancedOptionsString,
			}).Error("Error parsing a bool")
		}

		if advancedOptionsBool {
			// We should check our advanced options
			/////////////////////////////////////////////////////////////////////////////////////////////////////////
			/// Loopback option
			var advEnableLoopbackBool bool
			if advEnableLoopbackString, advEnableLoopbackOk := t.job.Parameters["adv_options_loopback"]; advEnableLoopbackOk {
				if advEnableLoopbackOk, err = strconv.ParseBool(advEnableLoopbackString); err != nil {
					log.WithFields(log.Fields{
						"error":      err,
						"boolString": advEnableLoopbackString,
					}).Error("Error parsing a bool")
				}

				if advEnableLoopbackBool {
					opts = append(opts, "--loopback")
				}
			}

			/////////////////////////////////////////////////////////////////////////////////////////////////////////
			/// Markov Options
			if advMarkovThresholdString, advMarkovThresholdOk := t.job.Parameters["adv_options_markov"]; advMarkovThresholdOk {
				if advMarkovThresholdInt, err := strconv.Atoi(advMarkovThresholdString); err != nil {
					if advMarkovThresholdInt <= 0 {
						log.WithField("adv_options_markov", advMarkovThresholdString).Error(err)
						return nil, err
					}

					opts = append(opts, "--markov-threshold="+advMarkovThresholdString)
				}
			}

			/////////////////////////////////////////////////////////////////////////////////////////////////////////
			/// Timeout Options
			if advTimeoutString, advTimeoutOk := t.job.Parameters["adv_options_timeout"]; advTimeoutOk {
				if advTimeoutInt, err := strconv.Atoi(advTimeoutString); err != nil {
					if advTimeoutInt <= 0 {
						log.WithField("adv_options_timeout", advTimeoutString).Error(err)
						return nil, err
					}

					opts = append(opts, "--runtime="+advTimeoutString)
				}
			}
		}
	}

	// Setup the output file argument
	opts = append(opts, "--outfile", filepath.Join(t.wd, HASH_OUTPUT_FILENAME))

	// Append args from the configuration file
	t.start = append(t.start, config.Args...)
	t.showPot = append(t.showPotLeft, "--hash-type="+htype, "--separator", ":")
	if config.PotFilePath != "" {
		t.showPot = append(t.showPot, "--potfile-path", config.PotFilePath)
	}

	// Setup the start and resume options
	t.start = append(t.start, "--session="+t.job.UUID)
	t.resume = append(t.resume, "--session="+t.job.UUID, "--restore")

	// Setup the show command for the showPot execution
	leftFilePath := filepath.Join(t.wd, HASHCAT_LEFT_FILENAME)
	t.showPotLeft = append(t.showPot, "--outfile", leftFilePath, "--left", USER_HASHES_FILENAME)

	showPotFilePath := filepath.Join(t.wd, HASHCAT_POT_SHOW_FILENAME)
	t.showPot = append(t.showPot, "--outfile", showPotFilePath, "--show", USER_HASHES_FILENAME)

	// Append the various inputs to the argument
	args = append(args, opts...)
	args = append(args, argHash, argDmD)

	// Apply the args parsed from the parameters
	t.start = append(t.start, args...)

	// Setup the OutputTitles column headers
	t.job.OutputTitles = []string{"Plaintext", "Hashes"}

	// Let's now get rid of the large parameter values we now have locally
	delete(t.job.Parameters, "dict_custom_prepend")
	delete(t.job.Parameters, "dict_rules_custom_file")
	delete(t.job.Parameters, "hashes_file_upload")
	delete(t.job.Parameters, "hashes_multiline")
	delete(t.job.Parameters, "dict_custom_prepend")

	return &t, nil
}
