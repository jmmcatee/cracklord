package directconnectresourcemanager

import (
	"crypto/tls"
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/emperorcow/protectedmap"
	"github.com/jmmcatee/cracklord/common/queue"
)

type resourceInfo struct {
	Notes         string
	ReconnectTime int
}

type directResourceManager struct {
	resources protectedmap.ProtectedMap
	q         *queue.Queue
}

func Setup(qpointer *queue.Queue) queue.ResourceManager {
	return &directResourceManager{
		resources: protectedmap.New(),
		q:         qpointer,
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

func (this directResourceManager) Parameters() string {
	return `"form": [
	    {
	        "type": "section",
	        "htmlClass": "row",
	        "items": [
	            {
	                "type": "section",
	                "htmlClass": "col-xs-6",
	                "items": [
	                    "reconnect"
	                ]
	            },
	            {
	                "type": "section",
	                "htmlClass": "col-xs-6",
	                "items": [
	                    {
	                        "type": "conditional",
	                        "condition": "modelData.reconnect",
	                        "items": [
	                            "reconnecttime"
	                        ]
	                    }
	                ]
	            }
	        ]
	    },
	    {
	        "key": "notes",
	        "type": "textarea",
	        "placeholder": "OPTIONAL: Any notes you would like to include (location, primary contact, etc.)"
	    }
	],
	"schema": {
		"type": "object",
		"title": "Direct Connect",
		"properties": {
		    "notes": {
			    "title": "Notes",
			    "type": "string"
		    },
		    "reconnect": {
	            "title": "Attempt automatic reconnect?",
	            "type": "boolean",
	            "default": true
	        },
	        "reconnecttime": {
	            "title": "Reconnect Time",
	            "description": "In seconds",
	            "type": "integer",
	            "default": 10
	        }
		}
	}`
}

func (this *directResourceManager) AddResource(name string, address string, params map[string]string, tls *tls.Config) error {
	//First, we attempt to add the resource into the queue itself
	uuid, err := this.q.AddResource(address, name, tls)

	//If unable to connect, log it and return the error to the API
	if err != nil {
		log.WithField("error", err.Error()).Error("Unable to add resource through direct connect manager")
		return err
	}

	//Let's create a temporary resource to hold the info
	tempresource = resourceInfo{
		ReconnectTime: -1,
		Notes:         params[notes],
	}

	//If we were going to try and reconnect (from the boolean parameter), then set the time to the value.
	if params[reconnect] == true {
		tempresource.ReconnectTime = params[reconnecttime]
	}

	//Finally, set the resource into our map
	this.resources.Set(uuid, tempresource)

	return nil
}

func (this *directResourceManager) DeleteResource(resourceid string) error {
	//First, try and delete the resource from the queue itself
	err := this.q.RemoveResource(resourceid)

	//If there was an error, log it back to the API
	if err != nil {
		log.WithField("error", err.Error()).Error("Unable to remove resource through direct connect manager")
		return err
	}

	//Finally, delete the local data from here
	this.resources.Delete(resourceid)
	return nil
}

func (this directResourceManager) GetResource(resourceid string) (queue.Resource, map[string]string, error) {
	return queue.Resource{}, nil
}

func (this *directResourceManager) PauseResource(resourceid string) error {
	return nil
}

func (this *directResourceManager) ResumeResource(resourceid string) error {
	return nil
}

func (this directResourceManager) GetManagedResources() []string {
	return []string{"one", "two"}
}

func (this *directResourceManager) Keep() {
	return
}
