package cracklog

import (
	"bytes"
	"fmt"
	"github.com/Sirupsen/logrus"
	"os"
	"path"
	"runtime"
	"sort"
)

//Struct to hold our logrus hook
type fileHook struct {
	Writer   *os.File
	Filename string
}

//The function that we'll use to pass the file location into this hook
func NewFileHook(location string) (*fileHook, error) {
	//Open our log file
	logwriter, err := os.OpenFile(location, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	return &fileHook{
		logwriter,
		location,
	}, err
}

//Function that will be called every time we have an item to log to the file
func (hook *fileHook) Fire(entry *logrus.Entry) error {
	//Prep a buffer to hold the output line
	b := &bytes.Buffer{}

	//First, let's put the time and level of the event
	fmt.Fprintf(b, "[%-39s %5s]", entry.Time.String(), entry.Level.String())

	//Now let's get the calling function, it should be 4 function calls (this fire, 2 in entry, and then debug/info/error/etc.)
	_, file, line, ok := runtime.Caller(4)
	if ok {
		fmt.Fprintf(b, " (%s:%d)", path.Base(file), line)
	}

	//Print the log message to the buffer
	fmt.Fprintf(b, " %s", entry.Message)

	//Pad everything with some spaces so our variables line up, make it easy to read
	for i := b.Len(); i < 140; i++ {
		b.WriteByte(' ')
	}

	//Get the keys for the labels we need to print, build an array out of them, then sort it
	var keys []string = make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	//Add the variables to our string
	for _, k := range keys {
		v := entry.Data[k]
		fmt.Fprintf(b, " %s=%v", k, v)
	}

	//Newlines are awesome
	b.WriteByte('\n')

	//Write it to the file and then return
	_, err := hook.Writer.WriteString(b.String())
	return err
}

//We want our hook to log all levels.  Note, this will still be changed by 
// level is set on the global log object
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
