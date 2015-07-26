package aws

import (
	"crypto/tls"
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/emperorcow/protectedmap"
	"github.com/jmmcatee/cracklord/common/queue"
	"time"
)

type resourceInfo struct {
	key         ec2.KeyPairInfo
	privatekey  string
	instance    ec2.Instance
	state       ec2.InstanceState
	starttime   time.Time
	lastusetime time.Time
}

type awsResourceManager struct {
	resources protectedmap.ProtectedMap
	q         *queue.Queue
	tls       *tls.Config
}

func Setup(qpointer *queue.Queue, tlspointer *tls.Config) queue.ResourceManager {
	return &awsResourceManager{
		resources: protectedmap.New(),
		q:         qpointer,
		tls:       tlspointer,
	}
}

func (this awsResourceManager) SystemName() string {
	return "aws"
}

func (this awsResourceManager) DisplayName() string {
	return "Amazon Web Services"
}

func (this awsResourceManager) Description() string {
	return "Spawn resources inside Amazon Web Services (AWS) for use. Instances can be automatically terminated if unused for a time."
}

func (this awsResourceManager) ParametersForm() string {
	/* TODO:  Build form
	 */
}

func (this awsResourceManager) ParametersSchema() string {
	/* TODO:  Build form
	 */

}

func (this *awsResourceManager) AddResource(params map[string]string) error {
	/* TODO: Steps for add resource
	1. Create new instance in AWS
	2. Wait until instance has a state of running
	3. Once running, attempt to connect to resource
	4. Retry connecting every 60 seconds until connected
	*/
}

func (this *awsResourceManager) DeleteResource(resourceid string) error {
	/* TODO: steps to disconnect / delete
	1. Disconnect the RPC connection gracefully if we can
	2. Remove resource from queue
	3. Terminate the instance
	*/
}

func (this awsResourceManager) GetResource(resourceid string) (*Resource, map[string]string, error) {
	/* TODO: There shouldn't be anything here that happens with the AWS SDK.
	This is just returning the data to the cracklord API, as such it'll be returning
	some of the data, still need to decide what data goes.
	*/
}

func (this *awsResourceManager) UpdateResource(resourceid string, newstatus string, newparams map[string]string) error {
	/* There won't really be anything to update for the AWS instances */
}

func (this awsResourceManager) GetManagedResources() []string {
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

func (this awsResourceManager) Keep() {
	/* TODO:  This function will need to do several things:
	1. Check the AWS status of a resource and get the InstanceState, if it's not running inform the queue
	2. See if a resource is in use, if so update the last used time
	*/
}
