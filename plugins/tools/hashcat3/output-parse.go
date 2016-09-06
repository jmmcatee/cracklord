package hashcat3

import (
	"bufio"
	log "github.com/Sirupsen/logrus"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
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
}

// StatusTable is a table to convert status numbers in hashcat to a word
var StatusTable = map[string]string{
	"0":  "Init",
	"1":  "Starting",
	"2":  "Running",
	"3":  "Paused",
	"4":  "Exhausted",
	"5":  "Cracked",
	"6":  "Aborted",
	"7":  "Quit",
	"8":  "Bypass",
	"9":  "StopAtCheckpoint",
	"10": "Autotune",
}

// ParseMachineOutput returns a Status for a given status line
func ParseMachineOutput(out string) Status {
	log.WithField("status2Parse", out).Debug("Parsing machine output")

	if len(out) < 6 {
		// Empty stdout so return empty status
		return Status{}
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
	// Scan each word and begin populating our status
	var speedLoop bool
	var tempLoop bool
	for wordScanner.Scan() {
		log.WithField("line", wordScanner.Text()).Info("Line")
		// Status
		if strings.Compare(wordScanner.Text(), "STATUS") == 0 {
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

			wordScanner.Scan()
			speedMs, err := strconv.ParseFloat(wordScanner.Text(), 64)
			if err != nil {
				// We had an error parsing this so skip this word
				log.WithField("error", err).Error("Error parsing speed per ms.")
				continue
			}

			// Calculate the speed for this device
			status.Speed = append(status.Speed, speedCnt/speedMs*1000)
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
	}

	// Set the time estimate
	attemptsLeft := status.Keyspace - status.Attempted
	var totalSpeed float64
	log.WithField("speed", status.Speed).Info("Speed Divide by 0")
	for i := range status.Speed {
		totalSpeed += status.Speed[i]
	}

	totalSpeedInt64, _ := big.NewFloat(totalSpeed).Int64()
	duration := time.Duration(attemptsLeft/totalSpeedInt64) * time.Second

	log.WithField("Attempts Left", attemptsLeft).Info()

	log.WithField("Total Speed", totalSpeedInt64).Info()
	log.WithField("Duration", duration.String()).Info()

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

	return status
}

// ParseShowPotOutput takes the output of the hashcat --show command and returns a 2D array of hashes and cleartext values
func ParseShowPotOutput(stdout string) [][]string {
	stdout = strings.Replace(stdout, "\r ", "\n", -1)
	stdout = strings.Replace(stdout, " \r", "\n", -1)
	// We want to loop on each line, so build a reader
	lineScanner := bufio.NewScanner(strings.NewReader(stdout))

	var output [][]string
	for lineScanner.Scan() {
		// Check for the separator character
		if strings.Contains(lineScanner.Text(), "|") {
			// This is a hash line so we need to parse it
			rows := strings.Split(lineScanner.Text(), "|")
			if len(rows) == 2 {
				output = append(output, []string{rows[1], rows[0]})
			}
		}
	}

	return output
}

// ParseShowPotLeftOutput will return the hashes not found in the pot file
func ParseShowPotLeftOutput(stdout string) []string {
	stdout = strings.Replace(stdout, "\r ", "\n", -1)
	stdout = strings.Replace(stdout, " \r", "\n", -1)
	// We want to loop on each line, so build a reader
	lineScanner := bufio.NewScanner(strings.NewReader(stdout))

	var output []string
	for lineScanner.Scan() {
		// Check if the line is a known output or is likely a hash
		if !strings.Contains(lineScanner.Text(), "hashcat") &&
			!strings.Contains(lineScanner.Text(), "Counting") &&
			!strings.Contains(lineScanner.Text(), "Parsed") &&
			len(strings.TrimSpace(lineScanner.Text())) != 0 {
			output = append(output, lineScanner.Text())
		}

	}

	return output
}
