package hashcat5

import (
	"bufio"
	"io"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"errors"

	"bytes"

	log "github.com/Sirupsen/logrus"
)

// Status standard return status
type Status struct {
	Status          string
	Attempted       int64
	Keyspace        int64
	Progress        float64
	EstimateTime    string
	Speed           []float64 // speed in hashes per sec
	RecoveredHashes int64
	TotalHashes     int64
	Temperature     []int
	Rejected        int64
	Utilization     []int
}

// StatusTable is a table to convert status numbers in hashcat to a word
var StatusTable = map[string]string{
	"0":  "Init",
	"1":  "Autotune",
	"2":  "SelfTest",
	"3":  "Running",
	"4":  "Paused",
	"5":  "Exhausted",
	"6":  "Cracked",
	"7":  "Aborted",
	"8":  "Quit",
	"9":  "Bypass",
	"10": "Aborted_Checkpoint",
	"11": "Aborted_Runtime",
	"13": "Error",
}

// ParseMachineOutput returns a Status for a given status line
func ParseMachineOutput(out string) (Status, error) {
	//log.WithField("status2Parse", out).Debug("Parsing machine output")

	if len(out) < 6 {
		// Empty stdout so return empty status
		return Status{}, errors.New("Length of line entry is 0")
	}

	lineReader := strings.NewReader(out)
	lineScanner := bufio.NewScanner(lineReader)
	lineScanner.Split(bufio.ScanLines)

	// Scan the lines for each output
	var lastStatus string
	for lineScanner.Scan() {
		if len(lineScanner.Text()) > 7 {
			if strings.Compare(lineScanner.Text()[0:6], "STATUS") == 0 {
				// We have a status line, so save it until we know we are at the end of our stdout
				lastStatus = lineScanner.Text()
			}
		}
	}

	// We should now have the last STATUS line
	wordReader := strings.NewReader(lastStatus)
	wordScanner := bufio.NewScanner(wordReader)
	wordScanner.Split(bufio.ScanWords)

	var status Status
	var statusLineFound bool
	// Scan each word and begin populating our status
	var speedLoop bool
	var tempLoop bool
	var utilLoop bool
	for wordScanner.Scan() {
		log.WithField("line", wordScanner.Text()).Info("Line")
		// Status
		if strings.Compare(wordScanner.Text(), "STATUS") == 0 {
			// We found a status line
			statusLineFound = true

			wordScanner.Scan() // Get to value
			status.Status = StatusTable[wordScanner.Text()]
		}

		// Exec Runtime
		if strings.Compare(wordScanner.Text(), "EXEC_RUNTIME") == 0 {
			// We need to disable the speed loop now
			speedLoop = false
		}

		// SpeedLoop
		if speedLoop {
			// Get a parse both values
			speedCnt, err := strconv.ParseFloat(wordScanner.Text(), 64)
			if err != nil {
				// We had an error parsing this so skip this word
				log.WithField("error", err).Error("Error parsing speed count.")
				continue
			}

			// Dump the legacy 1000 value
			wordScanner.Scan()

			status.Speed = append(status.Speed, speedCnt)
		}

		// Speed
		if strings.Compare(wordScanner.Text(), "SPEED") == 0 {
			// We are now in the speed loop, so trigger that
			speedLoop = true
		}

		// CURKU
		if strings.Compare(wordScanner.Text(), "CURKU") == 0 {

		}

		// PROGRESS
		if strings.Compare(wordScanner.Text(), "PROGRESS") == 0 {
			wordScanner.Scan()
			completed, err := strconv.ParseFloat(wordScanner.Text(), 64)
			if err != nil {
				log.WithField("error", err).Error("Error parsing the completed portion of progress.")
				continue
			}

			wordScanner.Scan()
			total, err := strconv.ParseFloat(wordScanner.Text(), 64)
			if err != nil {
				log.WithField("error", err).Error("Error parsing the total portion of progress.")
				continue
			}

			status.Progress = (completed / total) * 100
			status.Attempted, _ = big.NewFloat(completed).Int64()
			status.Keyspace, _ = big.NewFloat(total).Int64()
		}

		// RECHASH
		if strings.Compare(wordScanner.Text(), "RECHASH") == 0 {
			wordScanner.Scan()
			cracked, err := strconv.ParseInt(wordScanner.Text(), 10, 64)
			if err != nil {
				log.WithField("error", err).Error("Error parsing the cracked hash number.")
				continue
			}
			status.RecoveredHashes = cracked

			wordScanner.Scan()
			total, err := strconv.ParseInt(wordScanner.Text(), 10, 64)
			if err != nil {
				log.WithField("error", err).Error("Error parsing the total hash number.")
				continue
			}
			status.TotalHashes = total
		}

		// RECSALT
		if strings.Compare(wordScanner.Text(), "RECSALT") == 0 {

		}

		if strings.Compare(wordScanner.Text(), "REJECTED") == 0 {
			tempLoop = false

			wordScanner.Scan()
			rej, err := strconv.ParseInt(wordScanner.Text(), 10, 64)
			if err != nil {
				log.WithField("error", err).Error("Error parsing rejected number")
				continue
			}

			status.Rejected = rej
		}

		// TEMP
		if tempLoop {
			temp, err := strconv.Atoi(wordScanner.Text())
			if err != nil {
				log.WithField("error", err).Error("Error parsing temperature number.")
				continue
			}

			status.Temperature = append(status.Temperature, temp)
		}

		// TEMP
		if strings.Compare(wordScanner.Text(), "TEMP") == 0 {
			tempLoop = true
		}

		if utilLoop {
			util, err := strconv.Atoi(wordScanner.Text())
			if err != nil {
				log.WithField("error", err).Error("Error parsing utilization number.")
				continue
			}

			status.Utilization = append(status.Utilization, util)
		}

		if strings.Compare(wordScanner.Text(), "UTIL") == 0 {
			utilLoop = true
		}

	}

	// If we did not find a status line return a failure and nil status
	if !statusLineFound {
		return Status{}, errors.New("no status line found")
	}

	// Set the time estimate
	attemptsLeft := status.Keyspace - status.Attempted
	log.WithField("Attempts Left", attemptsLeft).Info()

	var totalSpeed float64
	for i := range status.Speed {
		totalSpeed += status.Speed[i]
	}

	totalSpeedInt64, _ := big.NewFloat(totalSpeed).Int64()
	if totalSpeedInt64 != 0 {
		duration := time.Duration(attemptsLeft/totalSpeedInt64) * time.Second

		estHours := int64(math.Floor(duration.Hours()))
		estMinutes := int64(math.Floor(duration.Minutes()))
		estSeconds := int64(math.Floor(duration.Seconds()))

		days := estHours / 24
		remainderHours := estHours % 24

		estMinutes = estMinutes % 60
		estSeconds = estSeconds % 60

		estDayString := strconv.FormatInt(days, 10)
		estHourString := strconv.FormatInt(estHours, 10)
		estMinutesString := strconv.FormatInt(estMinutes, 10)
		estSecondsString := strconv.FormatInt(estSeconds, 10)

		if estHours > 24 {
			estHourString := strconv.FormatInt(remainderHours, 10)

			status.EstimateTime = estDayString + "days " + estHourString + "h " + estMinutesString +
				"m " + estSecondsString + "s"
		} else if estHours > 0 {
			status.EstimateTime = estHourString + "h " + estMinutesString + "m " + estSecondsString + "s"
		} else if estHours <= 0 {
			status.EstimateTime = estMinutesString + "m " + estSecondsString + "s"
		} else if estHours <= 0 {
			status.EstimateTime = estSecondsString + "s"
		}

		log.WithField("Total Speed", totalSpeedInt64).Info()
		log.WithField("Duration", duration.String()).Info()
	} else {
		status.EstimateTime = "Unknown"
	}

	return status, nil
}

// ParseShowPotFile pull the line count and the hash output from the show pot outputfile
func ParseShowPotFile(r io.Reader, leftSplit int, hashMode string) (count int64, hashes [][]string) {
	fileLineScanner := bufio.NewScanner(r)

	for fileLineScanner.Scan() {
		count++

		// Count the splits in the output
		potSplit := bytes.Count(fileLineScanner.Bytes(), []byte(":"))

		// user:444:lmhash:ntlmhash:::						lp = 6
		//   0   1     2      3     4 5 6     7
		// user:444:lmhash:ntlmhash: : : :PasswordOne		ps = 7
		//   0   1     2      3     4 5 6     7    8  9
		// user:444:lmhash:ntlmhash: : : :Password: :One	ps = 9

		// The splits in the pot file output should be leftSplit + 1 or greater
		// Let's check just in case something bad is happening
		if leftSplit < potSplit {
			// At worst there should be 1 extra : (hash:pass) so this is a min req
			// Let's get the hash
			var hash []byte
			parts := bytes.Split(fileLineScanner.Bytes(), []byte(":"))
			for i := 0; i < len(parts); i++ {
				hash = append(hash, parts[i]...)

				// are we done?
				if i == leftSplit {
					// We are done
					break
				}

				// Append a separator
				if i < leftSplit {
					hash = append(hash, byte(':'))
				}
			}

			// Let's get the password
			// add the separator between the hash and password
			prefix := append(hash, byte(':'))
			password := bytes.TrimPrefix(fileLineScanner.Bytes(), prefix)

			// Add the password and hash to the output
			//output = append(output, []string{password, lineHash})

			// We have an edge case to deal with
			if leftSplit == 6 {
				switch hashMode {
				case "1000":
					// PWDUMP NTLM
					hashes = append(hashes, []string{string(password), strings.Split(string(hash), ":")[3]})
				case "3000":
					// PWDUMP LM
					hashes = append(hashes, []string{string(password), strings.Split(string(hash), ":")[2]})
				default:
					hashes = append(hashes, []string{string(password), string(hash)})
				}
			} else {
				hashes = append(hashes, []string{string(password), string(hash)})
			}

		} else {
			// For some reason we do not have the right split so log it and move on
			log.WithFields(
				log.Fields{
					"leftSplit":            leftSplit,
					"potSplit":             potSplit,
					"currentPotOutputLine": fileLineScanner.Text(),
				}).Info("Bad pot file line.")
		}
	}

	return
}

// ParseLeftHashFile takes an io.Reader and returns the number of lines (hashes)
// and the number of separators (:)
func ParseLeftHashFile(r io.Reader) (count int64, split int) {
	fileLineScanner := bufio.NewScanner(r)

	fileLineScanner.Scan()
	count++
	split = bytes.Count(fileLineScanner.Bytes(), []byte(":"))

	for fileLineScanner.Scan() {
		count++
	}

	return
}

// ParseHashcatOutputFile parses the Hashcat Output file
func ParseHashcatOutputFile(r io.Reader, inputSplit int, hashMode string) (count int64, hashes [][]string) {
	// We have some edge cases to deal with PWDUMP[NTLM]/PWDUMP[LM]/PASSWD/SHADOW
	switch hashMode {
	case "1000", "3000":
		// PWDUMP so flip to 0 for NTLM only output
		return ParseShowPotFile(r, 0, hashMode)
	default:
		return ParseShowPotFile(r, inputSplit, hashMode)
	}
}
