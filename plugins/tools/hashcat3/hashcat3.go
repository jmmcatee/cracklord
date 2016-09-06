package hashcat3

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/vaughan0/go-ini"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

// Config is structure to hold configuration
type Config struct {
	BinPath      string
	WorkingDir   string
	Args         []string
	Separator    string
	PotFilePath  string
	HashModes    HashModes
	Dictionaries Dictionaries
	RuleFiles    RuleFiles
	Charsets     Charsets
}

var config Config

// Setup configures this plugin for running and returns and error something is wrong.
func Setup(confPath string) error {
	log.Debug("Setting up hashcat 3.x plugin...")

	// Load the configuration file
	confFile, err := ini.LoadFile(confPath)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
			"file":  confPath,
		}).Error("Unable to load configuration file.")
		return err
	}

	// Get basic options for the binPath, WorkingDir
	basicConfig := confFile.Section("Basic")
	if len(basicConfig) == 0 {
		// Nothing retrieved, so return error
		log.Error(`No "Basic" configuration section.`)
		return errors.New(`No "Basic" configuration section.`)
	}

	// Setup BinPath & WorkingDir
	config.BinPath = basicConfig["binPath"]
	config.WorkingDir = basicConfig["workingdir"]

	log.WithFields(log.Fields{
		"binpath": config.BinPath,
		"WorkDir": config.WorkingDir,
	}).Debug("BinPath and WorkingDir")

	// Get the dictionary section
	dicts := confFile.Section("Dictionaries")
	if len(dicts) == 0 {
		// Nothing retrieved, so return error
		log.Error(`No "Dictionaries" configuration section.`)
		return errors.New(`No "Dictionaries" configuration section.`)
	}
	for key, value := range dicts {
		log.WithFields(log.Fields{
			"name": key,
			"path": value,
		}).Debug("Added dictionary")

		config.Dictionaries = append(config.Dictionaries, Dictionary{Name: key, Path: value})
	}
	sort.Sort(config.Dictionaries)

	// Get the rule section
	rules := confFile.Section("Rules")
	if len(rules) == 0 {
		// Nothing retrieved, so return error
		log.Error(`No "Rules" configuration section.`)
		return errors.New(`No "Rules" configuration section.`)
	}

	for key, value := range rules {
		log.WithFields(log.Fields{
			"name": key,
			"path": value,
		}).Debug("Added rule")

		config.RuleFiles = append(config.RuleFiles, RuleFile{Name: key, Path: value})
	}
	sort.Sort(config.RuleFiles)

	// Store the character sets configured for brute forcing in the config file
	charset := confFile.Section("BruteCharset")
	if len(charset) == 0 {

		// Nothing retrieved, so return error
		log.Error(`No "charset" configuration section.`)
		return errors.New(`No "charset" configuration section.`)
	}

	for key, value := range charset {
		log.WithFields(log.Fields{
			"name": key,
			"path": value,
		}).Debug("Added charset to hashcat")

		config.Charsets = append(config.Charsets, Charset{Name: key, Mask: value})
	}
	sort.Sort(config.Charsets)

	// Get additional options for use with hashcat
	options := confFile.Section("Options")
	if len(options) == 0 {
		// Nothing retrieved, so return error
		log.Error(`No options configuration section.`)
		return errors.New(`No options configuration section.`)
	}

	for flag, value := range options {
		log.WithFields(log.Fields{
			"flag":  flag,
			"value": value,
		}).Debug("Added option to hashcat")

		// Catch some important flags that we need later
		switch flag {
		case "--separator":
			config.Separator = value
		case "--potfile-path":
			config.PotFilePath = value
		}

		if value == "" {
			// We have a boolean flag so only add the flag
			config.Args = append(config.Args, flag)
		} else {
			// Append both the flag and its value
			config.Args = append(config.Args, flag, value)
		}
	}

	// Get ExcludeHashMode to drop some Hash Modes
	exHMMap := map[string]string{}
	excludeHashModes := confFile.Section("ExcludeHashMode")
	if len(excludeHashModes) == 0 {
		// Nothing retrieved, so return error
		log.Error(`No excludeHashModes configuration section.`)
		return errors.New(`No excludeHashModes configuration section.`)
	}

	for mode, name := range excludeHashModes {
		log.WithFields(log.Fields{
			"mode": mode,
			"name": name,
		}).Debug("Added excludeHashModes to hashcat")

		exHMMap[mode] = name
	}

	// Get hashcat help page
	help, err := exec.Command(config.BinPath, "--help").Output()
	if err != nil {
		// Something is wrong with our executable so log and fail
		log.WithField("error", err.Error()).Error("Error executing hashcat for help screen.")
		return err
	}

	// Get the hash modes table
	hashModesTable := HashcatHelpScanner(string(help), "Hash modes")

	for index, value := range hashModesTable["#"] {
		if exName, ok := exHMMap[value]; ok {
			log.WithField("ExcludedHashMode", exName).Debug("Excluded a Hash Mode")
			continue
		}

		category := hashModesTable["Category"][index]
		name := hashModesTable["Name"][index]

		config.HashModes = append(config.HashModes, HashMode{
			Number:   value,
			Name:     name,
			Category: category,
		})

	}

	log.Info("Hashcat 3.x tool successfully setup")

	return nil
}

// Helper functions

func createWorkingDir(workdir, uuid string) (string, error) {
	// Build a working directory for this job
	fullpath := filepath.Join(workdir, uuid)
	err := os.Mkdir(fullpath, 0700)

	if err != nil {
		// Couldn't make a directory so kill the job
		return "", errors.New("Unable to create working directory: " + err.Error())
	}
	log.WithField("path", fullpath).Debug("Tool (hashcat): Working directory created")

	return fullpath, nil
}
