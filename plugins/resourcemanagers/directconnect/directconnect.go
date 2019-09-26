package directconnectresourcemanager

import (
	"crypto/tls"
	"errors"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/emperorcow/protectedmap"
	"github.com/jmmcatee/cracklord/common/queue"
	"github.com/vaughan0/go-ini"
)

type resourceInfo struct {
	notes         string
	lastGoodCheck time.Time
}

type directResourceManager struct {
	resources protectedmap.ProtectedMap
	q         *queue.Queue
	tls       *tls.Config
}

func Setup(confPath string, qpointer *queue.Queue, tlspointer *tls.Config) (queue.ResourceManager, error) {
	dc := directResourceManager{
		resources: protectedmap.New(),
		q:         qpointer,
		tls:       tlspointer,
	}

	log.Debug("Setting up the direct connect resource manager")

	// Load the configuration file from the path provided during the setup function
	confFile, err := ini.LoadFile(confPath)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
			"file":  confPath,
		}).Error("Unable to load configuration file for direct resource manager.")
		return &directResourceManager{}, err
	}

	// Get the hosts section of the config (if it exists, it's optional)
	confHosts := confFile.Section("Hosts")
	if confHosts == nil || len(confHosts) == 0 {
		// If it doesn't exist, that's fine, let's just put an info message out there so the user knows it's possible
		log.Info("No hosts pre-configured for directconnect resource manager, moving on.")
	} else {
		// If it's configured, we'll loop through every key/value pair.  They should be in the format address=name (k=v)
		for k, v := range confHosts {
			// We'll loop through each host and connect to it on startup.
			log.WithFields(log.Fields{
				"address": k,
				"name":    v,
			}).Debug("Attempting to connect to resource from direct connect configuration file")

			// Perform the connection attempt
			err := dc.AddResource(map[string]string{
				"address": k,
				"name":    v,
			})

			// If it fails, we shouldn't error out the queue, but we should log an error
			if err != nil {
				log.WithFields(log.Fields{
					"address": k,
					"name":    v,
					"msg":     err,
				}).Error("Unable to conenct to resource in directconnect configuration file.")
			}
		}
	}

	return &dc, nil
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
		"name",
		"address",
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
	uuid, err := this.q.AddResource(name)
	if err != nil {
		return err
	}

	//Now we connect to the resource, and then let the user know the status
	err = this.q.ConnectResource(uuid, address, this.tls)
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
	parameters["notes"] = localres.notes

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
		case "running": //If our new status is resume, then resume the resource
			err = this.q.ResumeResource(resourceid)
			if err != nil {
				return err
			}
			break

		case "paused": //If the new status is pause, pause the resource in the queue
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
	log.Debug("Direct connect keeper starting up")
	iter := this.resources.Iterator()
	for data := range iter.Loop() {
		logger := log.WithField("resourceid", data.Key)
		logger.Debug("Gathering data on resource")
		localResource := data.Val.(resourceInfo)
		queueResource, ok := this.q.GetResource(data.Key)

		if !ok {
			logger.Error("Unable to find a resource in the queue that the direct connect manager thought it was responsible for.")
			continue
		}

		status := this.q.CheckResourceConnectionStatus(queueResource)
		logger.WithField("status", status).Debug("Checked resource connection status")

		logger.WithFields(log.Fields{
			"notes":        localResource.notes,
			"lastgoodtime": localResource.lastGoodCheck,
		}).Debug("Processing resource.")

		//If the connection to the resource is still good, let's flag when we last checked that
		//otherwise, we'll want to see about reconnecting
		if status {
			localResource.lastGoodCheck = time.Now()
		} else {
			logger.Debug("attempting resource reconnect")
			this.q.ReconnectResource(data.Key, this.tls)
		}

		//Update our local data for the resource
		this.resources.Set(data.Key, localResource)
	}

	log.Info("Direct connect resource manager has successfully updated resources.")
}

func (this *directResourceManager) parseParams(params map[string]string) resourceInfo {
	//Let's create a temporary resource to hold the info
	tempresource := resourceInfo{
		notes: params["notes"],
	}
	return tempresource
}
