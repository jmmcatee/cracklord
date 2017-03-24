package queue

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/emperorcow/protectedmap"
	"github.com/jmmcatee/cracklord/common"	
)

func OnJobCreate(job common.Job) {
	log.Debug("OnJobCreate functions starting.");

	var data HookJob

	data.ID = job.UUID
	data.Name = job.Name
	data.Status = job.Status
	data.Owner = job.Owner
	data.StartTime = job.StartTime
	data.CrackedHashes = job.CrackedHashes
	data.TotalHashes = job.TotalHashes
	data.Progress = job.Progress
	data.Params = job.Parameters
	data.ToolID = job.ToolUUID
	data.PerformanceTitle = job.PerformanceTitle
	data.PerformanceData = job.PerformanceData
	data.OutputTitles = job.OutputTitles
	data.OutputData = job.OutputData

	performWebPOST()
}

func OnJobStart(j common.Job) {

}

func OnJobFinish(j common.Job) {

}

func OnResourceConnect(r common.Resource) {

}

func OnQueueChange(q common.Queue) {

}

func performWebPOST(url string, data interface{}) error {
	b := new(bytes.Buffer);
	json.NewEncoder(b).Encode(data);

	log.Debug(fmt.sprintf("POSTing to webhook %s.")

	res, err := http.Post(url, "application/json; charset=utf-8", b);
	return err;
}