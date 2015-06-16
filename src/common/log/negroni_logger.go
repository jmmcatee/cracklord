package cracklog

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/negroni"	
	"net/http"
	"time"
)

type NegroniLogger struct { }

func NewNegroniLogger() *NegroniLogger {
	return &NegroniLogger{}
}

func (nl *NegroniLogger) ServeHTTP(rw http.ResponseWriter, r *http.Request, n http.HandlerFunc) {
	start := time.Now()
	n(rw, r)
	latency := time.Since(start)

	res := rw.(negroni.ResponseWriter)
	log.WithFields(log.Fields{
		"status":      res.Status(),
		"method":      r.Method,
		"remote":      r.RemoteAddr,
		"latency":     latency,
	}).Info(fmt.Sprintf("Handled request: %s", r.RequestURI))	
}