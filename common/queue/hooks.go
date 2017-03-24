package queue

/* The functions in this file handle the process of events fired from the queue.
 * These functions should always be called by value and in a goroutine to not
 * block important queue functions.  Each will then either run a script or
 * POST data to a webhook allow for functions to occur before and after
 * many events, such as Job creation or completion.
 */

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/jmmcatee/cracklord/common"
)

/* Runs on Job Creation
 */
func HookOnJobCreate(hooks []string, j common.Job) {
	log.WithField("id", j.UUID).Debug("Executing hooks against job creation.")

	data := copyJobToHookJob(j)

	hooksRun(hooks, data)
}

/* Runs on Job Start.  Note: This is only when a job is initially started,
 * and will NOT pickup on pauses and restarts.
 */
func HookOnJobStart(hooks []string, j common.Job) {
	log.WithField("id", j.UUID).Debug("Executing hooks against job start.")

	data := copyJobToHookJob(j)

	hooksRun(hooks, data)

}

/* Runs when a job finishes no matter the status
 */
func HookOnJobFinish(hooks []string, j common.Job) {
	log.WithField("id", j.UUID).Debug("Executing hooks against job finish.")

	data := copyJobToHookJob(j)

	hooksRun(hooks, data)

}

/* Runs when a resource is initially connected to the queue
 */
func HookOnResourceConnect(hooks []string, id string, r Resource) {
	log.WithField("name", r.Name).Debug("Executing hooks for resource connection")

	data := copyResourceToHookResource(r, id)

	hooksRun(hooks, data)
}

/* Runs when the queue is reordered
 */
func HookOnQueueReorder(hooks []string, stack []common.Job) {
	log.Debug("Executing hooks for queue reorder")

	var data HookQueueOrder

	for _, j := range stack {
		data.JobOrder = append(data.JobOrder, copyJobToHookQueueJob(j))
	}

	hooksRun(hooks, data)
}

/* Performs a web POST request against a specific URL.  Used to send the
 * relevant data to the webhook defined in our configuration files
 */
func hookPerformWebPOST(url string, data interface{}) error {
	// Generate a buffer for us to store some JSON
	b := new(bytes.Buffer)

	// Take the data we have received and encode it in JSON to POST
	json.NewEncoder(b).Encode(data)

	// It's always important to log.
	log.WithFields(log.Fields{
		"url": url,
	}).Debug("POSTing to webhook")

	// POST up our data and then return if we got an error or not.
	res, err := http.Post(url, "application/json; charset=utf-8", b)

	log.WithFields(log.Fields{
		"url":    url,
		"code":   res.StatusCode,
		"status": res.Status,
	}).Debug("Response received from webhook")

	return err
}

/* Not yet implemented.  TODO
 */
func hookPerformScriptExecute(path string, data interface{}) error {
	return nil
}

/* Takes a common Job type and concerts it into the struct type we have
 * defined for all job hooks.
 */
func copyJobToHookJob(src common.Job) (dst HookJob) {
	dst.ID = src.UUID
	dst.Name = src.Name
	dst.Status = src.Status
	dst.Owner = src.Owner
	dst.StartTime = src.StartTime
	dst.CrackedHashes = src.CrackedHashes
	dst.TotalHashes = src.TotalHashes
	dst.Progress = src.Progress
	dst.Params = src.Parameters
	dst.ToolID = src.ToolUUID
	dst.PerformanceTitle = src.PerformanceTitle
	dst.PerformanceData = src.PerformanceData
	dst.OutputTitles = src.OutputTitles
	dst.OutputData = src.OutputData

	return dst
}

/* Takes a common Resource type and concerts it to the hook resource output
 * type struct
 */
func copyResourceToHookResource(src Resource, uuid string) (dst HookResource) {
	dst.ID = uuid
	dst.Name = src.Name
	dst.Address = src.Address

	return dst
}

/* Takes a common Job type and converts it to the type we have defined for
 * queue hook data
 */
func copyJobToHookQueueJob(src common.Job) (dst HookQueueOrderJobData) {
	dst.ID = src.UUID
	dst.Name = src.Name
	dst.Status = src.Status
	dst.Owner = src.Owner
	dst.StartTime = src.StartTime
	dst.CrackedHashes = src.CrackedHashes
	dst.TotalHashes = src.TotalHashes
	dst.Progress = src.Progress

	return dst
}

/* This function looks at the hook to determine if it's a webhook or not.
 * For now this will return a string with the type "web" or "script"
 */
func hookParseType(target string) string {
	if strings.HasPrefix(target, "http") {
		return "web"
	} else {
		return "script"
	}
}

/* This function runs the hooks that have been passed into it utilizing the data
 * passed in as well.  We will need to look through all of the registered hooks
 * and then execute them
 */
func hooksRun(hooks []string, data interface{}) {
	for _, h := range hooks {
		switch hookParseType(h) {
		case "web":
			hookPerformWebPOST(h, data)
		case "script":
			hookPerformScriptExecute(h, data)
		}
	}
}
