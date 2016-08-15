package common

import (
	"io"
	"os"
	"strings"
)

// Copy a file from one place to another
func Copy(dst, src string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}

	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}

	return d.Close()
}

// CopyPrepend a file with prepended value
func CopyPrepend(dst, src, prepend string) error {
	// Make the prepend string a buffer
	pr := strings.NewReader(prepend + "\n")

	s, err := os.Open(src)
	if err != nil {
		return err
	}

	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(d, pr); err != nil {
		d.Close()
		return err
	}

	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}

	return d.Close()
}
