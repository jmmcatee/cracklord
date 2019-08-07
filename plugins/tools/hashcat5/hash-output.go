package hashcat5

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
)

// DecodeBase64Upload decodes Base64 file uploads
func decodeBase64Upload(fileUpload string) ([]byte, error) {
	// Split the file uploads into the various parts
	fileParts := strings.Split(fileUpload, ";")
	if len(fileParts) != 3 {
		err := errors.New("Error parsing the uploaded file")
		log.Error(err.Error())
		return []byte{}, err
	}
	// [0] - file:[filename of uploaded file]
	// [1] - data:[data type (text/plain)]
	// [2] - base64,[data]

	// Decode the base64 file content
	decodedBytes, err := base64.StdEncoding.DecodeString(fileParts[2][7:])
	if err != nil {
		log.WithField("error", err).Error("Error parsing the base64 file contents")
		return []byte{}, err
	}

	// Return the decoded bytes
	return decodedBytes, nil
}

// GetHashesFromBytesInput returns an array of []byte that are each hash uploaded
func getHashesFromBytesInput(input []byte, sep string) ([][]byte, int) {
	var hashes = [][]byte{}
	var count int

	// Create a line reading buffer
	lbuf := bytes.NewBuffer(input)
	lscan := bufio.NewScanner(lbuf)

	// Loop on each line and store in array of hashes
	for lscan.Scan() {
		var b []byte

		count = bytes.Count(lscan.Bytes(), []byte(":"))

		b = lscan.Bytes()

		hashes = append(hashes, b)
	}

	// return what we have
	return hashes, count
}

// WriteHashes2File writes hashes to a file
func writeHashes2File(hashes [][]byte, path string) error {
	// Open the hashes file
	file, err := os.Create(path)
	if err != nil {
		log.WithField("hashFile", path).Error(err.Error())
		return err
	}

	for i := range hashes {
		_, err = file.Write(hashes[i])
		if err != nil {
			log.WithField("hashFile", path).Error(err.Error())
			return err
		}

		if i < len(hashes)-1 {
			_, err = file.Write([]byte("\n"))
			if err != nil {
				log.WithField("hashFile", path).Error(err.Error())
				return err
			}
		}
	}

	// We should have written all the hashes at this point to close the file and return
	file.Close()
	return nil
}
