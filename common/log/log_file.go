package log_file

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"os"
)

type fileHook struct {
	Writer   *os.File
	Filename string
}

func NewFileHook(location string) (*fileHook, error) {
	logwriter, err := os.OpenFile(location, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	return &fileHook{
		logwriter,
		location,
	}, err
}

func (hook *fileHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read log entry for writing: %v", err)
		return err
	}

	num, err := hook.Writer.WriteString(line + "\n")
	if err != nil {
		return err
	}
	num++
	return nil

}

func (hook *fileHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}
