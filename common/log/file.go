package log_file

import (
	"bytes"
	"fmt"
	"github.com/Sirupsen/logrus"
	"os"
	"path"
	"runtime"
	"sort"
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
	b := &bytes.Buffer{}
	fmt.Fprintf(b, "[%-39s %5s]", entry.Time.String(), entry.Level.String())

	_, file, line, ok := runtime.Caller(4)
	if ok {
		fmt.Fprintf(b, " (%s:%d)", path.Base(file), line)
	}

	fmt.Fprintf(b, " %s", entry.Message)

	for i := b.Len(); i < 140; i++ {
		b.WriteByte(' ')
	}

	var keys []string = make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := entry.Data[k]
		fmt.Fprintf(b, " %s=%v", k, v)
	}

	b.WriteByte('\n')

	_, err := hook.Writer.WriteString(b.String())
	return err
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
