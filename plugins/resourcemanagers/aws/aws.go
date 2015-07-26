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

}

func (this awsResourceManager) ParametersSchema() string {

}

func (this *awsResourceManager) AddResource(params map[string]string) error {

}

func (this *awsResourceManager) DeleteResource(resourceid string) error {

}

func (this awsResourceManager) GetResource(resourceid string) (*Resource, map[string]string, error) {

}

func (this *awsResourceManager) UpdateResource(resourceid string, newstatus string, newparams map[string]string) error {

}

func (this awsResourceManager) GetManagedResources() []string {

}

func (this awsResourceManager) Keep() {

}
