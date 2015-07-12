package queue

import ()

/* The ResourceManager interface is used to implement functionality to manage
 * resources throughout the queue.  All resource actions (add, remove, update)
 * will need to go through a manager, even for default or physical resources.
 * This interface is what the queue will expect for all manager operations.
 */
type ResourceManager interface {
	//SystemName returns the string used internally in the queue as a key.  This
	//should only use characters that are valid inside a URL.
	SystemName() string
	//DisplayName returns a string for the name that should be presented to humans
	DisplayName() string
	//Description returns a string with a short description of the resource manager
	Description() string
	//Parameters returns a string containing a JSON object with the necessary
	//information to add a new resource using this manager
	Parameters() string
	//AddResource takes a map of input (as required by parameters), the name and
	//address of the resource.  Finally, it also takes a tls configuration that
	//will be required by the Queue to connect to the resource.  It should then
	//add the resource to the queue itself for use by jobs.  Can return an error
	//but that will be nil is there were no issues.
	AddResource(name string, address string, params map[string]string) error
	//DeleteResource will remove a resource from the queue as soon as it is able
	//pending job completion.  Takes a parameter of the resource UUID and will
	//return an error if there are any problems.
	DeleteResource(resourceid string) error
	//GetResource takes the UUID of a resource in the queue and will return all
	//of the information about it in a queue.Resource object, as well as the
	//parameters that the resourcemanager is tracking internally.  Can also return
	//an error if there are any problems.
	GetResource(resourceid string) (*Resource, map[string]string, error)
	//Update resource allows the API to change the status of an existing resource
	//or update the parameters internal to the resource manager.  Address, name,
	//and other queue related resource data cannot be changed.
	UpdateResource(resourceid string, newstatus string, newparams map[string]string) error
	//GetManagedResources returns a slice of all resource UUIDs managed by this
	//resource manager.
	GetManagedResources() []string
	//Keep is a function that will be run on a regular basis (timing depends on
	//queue configuration, job status, etc.) and should process any ongoing needs
	//of the resource manager.  For example, the physical manager may attempt to
	//reconnect lost resources, or a Cloud based manager will stop resources that
	//have not been used in a set period of time to save money. Will be run
	//inside the main queue keeper synchronously.
	Keep()
}
