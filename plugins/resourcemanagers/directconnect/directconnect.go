package directconnectresourcemanager

import (
	"crypto/tls"
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/emperorcow/protectedmap"
	"github.com/jmmcatee/cracklord/common/queue"
	"strconv"
)

type resourceInfo struct {
	Notes     string
	Reconnect bool
}

type directResourceManager struct {
	resources protectedmap.ProtectedMap
	q         *queue.Queue
	tls       *tls.Config
}

func Setup(qpointer *queue.Queue, tlspointer *tls.Config) queue.ResourceManager {
	return &directResourceManager{
		resources: protectedmap.New(),
		q:         qpointer,
		tls:       tlspointer,
	}
}

func (this directResourceManager) SystemName() string {
	return "directconnect"
}

func (this directResourceManager) DisplayName() string {
	return "Direct Connect"
}

func (this directResourceManager) Description() string {
	return "Directly connect to resource servers."
}

func (this directResourceManager) ParametersForm() string {
	return `[
			{
				"type": "section",
				"htmlClass": "row",
				"items": [
					{
						"type": "section",
						"htmlClass": "col-xs-6",
						"items": [
							"name"
						]
					},
					{
						"type": "section",
						"htmlClass": "col-xs-6",
						"items": [
							"address"
						]
					}
				]
			},
         "reconnect",
			{
				"key": "notes",
				"type": "textarea",
				"placeholder": "OPTIONAL: Any notes you would like to include (location, primary contact, etc.)"
			}
    	]`
}
func (this directResourceManager) ParametersSchema() string {
	return `{
		"type": "object",
		"title": "Direct Connect",
		"properties": {
			"name": {
				"title": "Name",
				"type": "string",
				"description": "The name you would like to reference this resource as."
			},
			"address": {
				"title": "Address",
				"type": "string",
				"default": "localhost",
				"description": "The full DNS name or IP address of the resource."
			},
			"notes": {
				"title": "Notes",
				"type": "string"
			},
			"reconnect": {
				"title": "Attempt automatic reconnect?",
				"type": "boolean",
				"default": true,
				"description": "Should the system attempt to automatically reconnect in the event this resource becomes disconnected?"
			}
		},
		"required": [
			"name",
			"address",
			"reconnect"
		]
	}`
}

func (this *directResourceManager) AddResource(params map[string]string) error {
	//First, we need to get the name and address out of the parameters, as we're getting those from the user in this resource manager
	address, ok := params["address"]
	if !ok {
		return errors.New("Cannot add resource, address was not specified.")
	}
	name, ok := params["name"]
	if !ok {
		return errors.New("Cannot add resource, name was not specified.")
	}

	//First, we attempt to add the resource into the queue itself
	uuid, err := this.q.AddResource(address, name, this.tls)

	//If unable to connect, log it and return the error to the API
	if err != nil {
		return err
	}

	//Finally, set the resource into our map
	this.resources.Set(uuid, this.parseParams(params))

	return nil
}

func (this *directResourceManager) DeleteResource(resourceid string) error {
	//First, try and delete the resource from the queue itself
	err := this.q.RemoveResource(resourceid)

	//If there was an error, log it back to the API
	if err != nil {
		log.WithField("error", err.Error()).Debug("Unable to remove resource through direct connect manager")
		return err
	}

	//Finally, delete the local data from here
	this.resources.Delete(resourceid)
	return nil
}

func (this directResourceManager) GetResource(resourceid string) (*queue.Resource, map[string]string, error) {
	//First, get the resource itself from the queue
	resource, ok := this.q.GetResource(resourceid)

	//If we weren't able to gather it, return an error
	if !ok {
		return &queue.Resource{}, nil, errors.New("Resource with requested ID not found in the queue.")
	}

	//Now we'll gather the data from our local map of parameters
	localresource, ok := this.resources.Get(resourceid)
	if !ok {
		return &queue.Resource{}, nil, errors.New("Resource with requested ID could not be found in direct connect resource manager.")
	}

	//Since our map uses a generic interface{}, we have to cast our result back
	localres := localresource.(resourceInfo)

	//Parse our parameters struct back into a common string map
	parameters := make(map[string]string)
	parameters["notes"] = localres.Notes
	parameters["reconnect"] = strconv.FormatBool(localres.Reconnect)

	return resource, parameters, nil
}

func (this *directResourceManager) UpdateResource(resourceid string, newstatus string, newparams map[string]string) error {
	//Because we need to make some comparisons for pause/resume, let's get the current resource state
	oldresource, _, err := this.GetResource(resourceid)
	if err != nil {
		return err
	}

	//Set the internal parameters within the direct connect manager to the new data
	this.resources.Set(resourceid, this.parseParams(newparams))

	//Check to see if the old status matches the new one, if not, we need to make a change
	if oldresource.Status != newstatus {
		switch newstatus {
		case "resume": //If our new status is resume, then resume the resource
			err = this.q.ResumeResource(resourceid)
			if err != nil {
				return err
			}
			break

		case "pause": //If the new status is pause, pause the resource in the queue
			err = this.q.PauseResource(resourceid)
			if err != nil {
				return err
			}
			break
		}
	}

	//Finally, we can return a nil as we were successful
	return nil
}

func (this directResourceManager) GetManagedResources() []string {
	//We need to make a slice of resource UUID strings for every resource we manage.  First, let's make the actual slice with a length of the size of our map
	resourceids := make([]string, this.resources.Count())

	//Next let's start up an iterator for our map and loop through each resource
	iter := this.resources.Iterator()
	for data := range iter.Loop() {
		//Now let's add the ID from the map to the slice of UUIDs
		resourceids = append(resourceids, data.Key)
	}

	return resourceids
}

//This function loops through all of the directly connected resources and detects
//that resource is still connected.  If so, it will do nothing; however, if not
//then it will attempt to reconnect if at all possible.
func (this *directResourceManager) Keep() {
	iter := this.resources.Iterator()
	for data := range iter.Loop() {
		localResource := data.Val.(resourceInfo)
		queueResource, ok := this.q.GetResource(data.Key)
		if !ok {
			log.WithField("resourceid", data.Key).Error("Unable to find a resource in the queue that the direct connect manager thought it was responsible for.")
			continue
		}
		status := this.q.CheckResourceConnectionStatus(queueResource)
		if !status && localResource.Reconnect {
			log.WithField("resource", queueResource.Name).Info("Attempting to reconnect to directly connected resource.")
			this.q.ConnectResource(queueResource, this.tls)
		}
	}

	log.Info("Direct connect resource manager has successfully updated resources.")
}

func (this *directResourceManager) parseParams(params map[string]string) resourceInfo {
	//Let's create a temporary resource to hold the info
	tempresource := resourceInfo{
		Reconnect: false,
		Notes:     params["notes"],
	}

	//Set the boolean parameter for reconnect if true.
	if params["reconnect"] == "true" {
		tempresource.Reconnect = true
	}

	return tempresource
}
