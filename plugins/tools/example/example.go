package exampleplugin

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/jmmcatee/cracklord/common"
	"github.com/vaughan0/go-ini"
)

/*
	Function to setup the
*/
func Setup(path string) error {
	log.Debug("Setting up example tool")

	confFile, err := ini.LoadFile(path)
	if err != nil {
		return err
	}

	basic := confFile.Section("Basic")
	if len(basic) == 0 {
		// Nothing retrieved, so return error
		return errors.New("No \"Basic\" configuration section.")
	}

	log.Info("Example tool successfully setup")
	return nil
}

/*
	Struct to hold the "tooler" information used by the resource server to setup
	the plugin and maintain it's state.  This is the information that allows the
	queue server to start jobs on the "tasker" portion.
*/
type exampleTooler struct {
	toolUUID string
}

/*
	Return the name of the plugin.  This will be presented to users via the
	API when they select a tool
*/
func (h *exampleTooler) Name() string {
	return "Example Tool Plugin"
}

/*
	Return a string of the type of plugin.  This can be used via the API
	for categorization of tools.
*/
func (h *exampleTooler) Type() string {
	return "Example"
}

/*
	Return the version of the tool.  In the event multiple versions of the same
	tool exist, all will be presented to the API.  Tools with the same name
	and version will only be presented once.  Jobs can only be started on one
	tool (name + version) at a time.
*/
func (h *exampleTooler) Version() string {
	return "0.0"
}

/*
	Return the UUID of this tool.  Note, if the same tool is running on multiple
	resources they may have different UUIDs, this is expected behavior, which is
	why version and name are used to determine duplicates of tools
*/
func (h *exampleTooler) UUID() string {
	return h.toolUUID
}

/*
	Set the UUID of the tool as necessary
*/
func (h *exampleTooler) SetUUID(s string) {
	h.toolUUID = s
}

/*
	Return information about what data is required to start a job utilizing this
	tool.  The first UI for CrackLord was written using AngularJS, so the schema
	for this information used a common form schema.

	For information regarding the schema, see schemaform.io.  To test the output
	of a schema, you can visit: http://schemaform.io/examples/bootstrap-example.html

	The form is split into two JSON strings, one array and one object. The
	first is the form array, which contains a listing of the fields that will be
	in the form.  There are some pre-defined keywords (see schemaform.io docs),
	otherwise each field takes several properties for that array element.

	Second is the schema object, whcih defines the details and validation
	requirements for each of the items named in the form array.  The API and
	default web GUI will process this information and present the form to the
	user, the queue and resource servers simply see these as strings to be
	passed, allowing a great deal of flexibility in the form.
*/
func (h *exampleTooler) Parameters() string {
	params := `{
		"form": [
		],
		"schema": {
		} `

	return params
}

/*
	Return the type of resource that will be used by this tool.  Typically this
	will be either GPU or CPU; however, additional types can be configured
	in common.go as necessary.  The queue will manage jobs by sending them to
	resources with the necessary resources available.
*/
func (h *exampleTooler) Requirements() string {
	return common.RES_GPU
}

/*
	Start a new job by using the tasker for this tool
*/
func (h *exampleTooler) NewTask(job common.Job) (common.Tasker, error) {
	return newExampleTask(job)
}

/*
	Return the tooler object for proper setup
*/
func NewTooler() common.Tooler {
	return &exampleTooler{}
}
