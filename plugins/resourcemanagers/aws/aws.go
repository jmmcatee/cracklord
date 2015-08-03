package awsresourcemanager

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/emperorcow/protectedmap"
	"github.com/jmmcatee/cracklord/common"
	"github.com/jmmcatee/cracklord/common/queue"
	"github.com/vaughan0/go-ini"
	"sort"
	"strconv"
	"time"
)

type resourceInfo struct {
	Instance       ec2.Instance
	State          int64
	StartTime      time.Time
	LastUseTime    time.Time
	DisconnectTime time.Duration
}

type config struct {
	AccessKey          string
	AccessSecret       string
	Region             string
	AMIID              string
	VPCID              string
	SecurityGroup      string
	InstanceTypes      map[string]string
	InstanceTypesOrder []string
	CACert             *x509.Certificate
	CAKey              *rsa.PrivateKey
}

var conf = config{}

type awsResourceManager struct {
	resources     protectedmap.ProtectedMap
	q             *queue.Queue
	tls           *tls.Config
	lastAPIUpdate time.Time
	vpc           ec2.VPC
	secgrp        ec2.SecurityGroup
	subnets       []*ec2.Subnet
	ec2client     *ec2.EC2
}

func Setup(confpath string, qpointer *queue.Queue, tlspointer *tls.Config, caCertPath, caKeyPath string) (queue.ResourceManager, error) {
	log.Debug("Setting up AWS resource manager")

	// Load the configuration file from the path provided during the setup function
	confFile, err := ini.LoadFile(confpath)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
			"file":  confpath,
		}).Error("Unable to load configuration file for AWS resource manager.")
		return &awsResourceManager{}, err
	}

	// Get the bin path
	confGen := confFile.Section("General")
	if len(confGen) == 0 {
		// Nothing retrieved, so return error
		return &awsResourceManager{}, errors.New("No \"General\" configuration section.")
	}

	// Declare a boolean to hold if there are any issues.  Then go through each of the confiugration
	// lines and make sure we have everything in the "General" area that we are looking for.
	var ok bool
	conf.AccessKey, ok = confGen["AccessKeyID"]
	if !ok {
		return &awsResourceManager{}, errors.New("AccessKeyID was not found in the general configuration section of the AWS resource manager config")
	}
	conf.AccessSecret, ok = confGen["SecretAccessKey"]
	if !ok {
		return &awsResourceManager{}, errors.New("SecretAccessKey was not found in the general configuration section of the AWS resource manager config")
	}
	conf.Region, ok = confGen["Region"]
	if !ok {
		return &awsResourceManager{}, errors.New("The Region was not defined in the general configuration section of the AWS resource manager config")
	}
	conf.AMIID, ok = confGen["AMIID"]
	if !ok {
		return &awsResourceManager{}, errors.New("The AMIID of the image to deploy was not defined in the general configuration section of the AWS resource manager config")
	}
	conf.SecurityGroup, ok = confGen["SecurityGroupName"]
	if !ok {
		return &awsResourceManager{}, errors.New("The ID of the security group to use was not defined in the general configuration section of the AWS resource manager config")
	}

	// If we don't have a VPCID defined, we'll assume the default one.
	conf.VPCID, ok = confGen["VPCID"]
	if !ok {
		conf.VPCID = ""
	}

	// Get the InstanceTypes section
	confTypes := confFile.Section("InstanceTypes")
	if len(confTypes) == 0 {
		// Nothing retrieved, so return error
		return &awsResourceManager{}, errors.New("No 'InstanceTypes' configuration section in aws config.")
	}
	conf.InstanceTypes = make(map[string]string)
	for key, value := range confTypes {
		log.WithFields(log.Fields{
			"id":   key,
			"name": value,
		}).Debug("Added instance type to AWS resource manager configuration")
		conf.InstanceTypes[value] = key
	}
	conf.InstanceTypesOrder = getSortedKeys(conf.InstanceTypes)

	conf.CACert, conf.CAKey, err = common.GetCertandKey(caCertPath, caKeyPath)
	if err != nil {
		return &awsResourceManager{}, err
	}

	aws := awsResourceManager{
		resources: protectedmap.New(),
		q:         qpointer,
		tls:       tlspointer,
		ec2client: getEC2Client(conf.AccessKey, conf.AccessSecret, conf.Region),
	}

	aws.gatherAPIData()

	return &aws, nil
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
	form := `[
	{
		"key": "subnet",
		"type": "select",
		"titleMap": {`
	for i, v := range this.subnets {
		if i > 0 {
			form += ","
		}
		form += "\"" + *v.SubnetID + "\": \"" + *v.SubnetID + " (" + *v.CIDRBlock + ")\""
	}

	form += `}
	},
	{
		"type": "section",
		"htmlClass": "row",
		"items": [
			{
				"type": "section",
				"htmlClass": "col-xs-6",
				"items": [
					{
						"key": "disconnect",
						"type": "radiobuttons",
						"style": {
							"selected": "btn-success",
							"unselected": "btn-default"
						},
						"titleMap": [
							{
								"value": "true",
								"name": "Yes"
							},
							{
								"value": "false",
								"name": "No"
							}
		        	   ]
		        	}
				]
			},
			{
				"type": "section",
				"htmlClass": "col-xs-6",
				"items": [
					{
						"key": "disconnecttime",
						"condition": "model.disconnect == 'true'"
					}
				]
			}
		]
	},
	"instancetype",
	"number"
]`

	return form
}

func (this awsResourceManager) ParametersSchema() string {
	schema := `{
	"type": "object",
	"title": "Comment",
	"properties": {
		"subnet": {
			"title": "Subnet",
			"type": "string",
			"enum": [`
	for i, v := range this.subnets {
		if i > 0 {
			schema += ","
		}
		schema += "\"" + *v.SubnetID + "\""
	}
	schema += `
			],
			"default": "` + *this.subnets[0].SubnetID + `"
	   },
		"disconnect": {
			"title": "Terminate Host When Unused?",
			"description": "Should these instances be automatically disconnected to reduce costs?",
			"type": "string",
			"default": "true"
		},
		"disconnecttime": {
			"title": "Terminate Time",
			"description": "How many minutes to wait until unused instance is terminated",
			"type": "string",
			"default": "15"
		},
		"instancetype": {
			"title": "Instance Type",
			"type": "string",
			"enum": [`
	for i, v := range conf.InstanceTypesOrder {
		if i > 0 {
			schema += ","
		}
		schema += "\"" + v + "\""
	}
	schema += `
			]
		},
		"number": {
			"title": "Number of Instances",
			"description": "How many instances should be started and connected to CrackLord?",
			"default": "1",
			"type": "string"
		}
	},
	"required": [
		"subnet",
		"securitygroup",
		"instancetype",
		"disconnect",
		"number"
	]
}`

	return schema
}

/* This function does the following things to get a resource added.
1. Check our input from the form to make sure it's proper
2. Create the instance in AWS
3. Create a goroutine that checks every 60 seconds to see if the instance is in a ready state
*/
func (this *awsResourceManager) AddResource(params map[string]string) error {
	// First, we check all of our inputs and make sure they are correct, if not, we generate and return errors.
	tmpnum, ok := params["number"]
	if !ok {
		return errors.New("A number of instances was not specified.")
	}

	// Number should be an int
	num, err := strconv.Atoi(tmpnum)
	if err != nil {
		return err
	}

	subnet, ok := params["subnet"]
	if !ok {
		return errors.New("Subnet was not specified.")
	}

	typeKey, ok := params["instancetype"]
	if !ok {
		return errors.New("Instance type was not specified")
	}

	disconTime := -1
	disconnect, ok := params["disconnect"]
	if ok {
		if disconnect == "true" {
			tmptime, ok := params["disconnecttime"]
			if ok {
				disconTime, _ = strconv.Atoi(tmptime)
			}
		}
	}

	// Instancetype should be one of the ones we know about from our map
	instancetype, ok := conf.InstanceTypes[typeKey]
	if !ok {
		return errors.New("Instance type (" + typeKey + ") is unknown.")
	}

	// Gather the PEM format (in strings) of the CA cert, private key, and public cert.
	// These will be submitted in the user data and will end up being loaded into the
	// AMI as a certificate to authenticate the queue to the resource.
	cert, key, err := common.GenerateResourceKeys(conf.CACert, conf.CAKey, "*.*.compute.amazonaws.com")
	certString, err := common.WriteCertificateToString(cert)
	if err != nil {
		return err
	}

	keyString, err := common.WriteRSAPrivateKeyToString(key)
	if err != nil {
		return err
	}

	caCertString, err := common.WriteCertificateToString(conf.CACert)
	if err != nil {
		return err
	}

	// Now that we have all of our data, let's actually launch the instance in the API.
	res, err := launchInstance(conf.AMIID, *this.secgrp.GroupID, subnet, instancetype, caCertString, certString, keyString, num, this.ec2client)
	if err != nil {
		return errors.New("Unable to start instance: " + err.Error())
	}

	//Now we start a goroutine that waits for the instance to be in a ready state, and then we'll do the rest.
	//For now, let's return to the user that we're trying.
	for _, instance := range res.Instances {
		go this.waitForResourceReady(disconTime, *instance.InstanceID, this.ec2client)
	}

	return nil
}

// This function will be run in a goroutine and will check every 30 seconds to see
// if the instance we just started is ready.
func (this *awsResourceManager) waitForResourceReady(disconnect int, instanceid string, ec2client *ec2.EC2) {
	// Setup a ticker that will hit every 30 seconds
	ticker := time.NewTicker(30 * time.Second)

	// Loop forever, waiting for our ticker to hit every 30 seconds.  This loop
	// will check to see that our instance is finally in a running state, at which
	// point we'll add it to the queue.
	for {
		select {
		case <-ticker.C:
			//Lookup the state of the instance to determine if we're running yet
			state, err := getInstanceState(instanceid, ec2client)
			if err != nil {
				log.WithFields(log.Fields{
					"error":      err.Error(),
					"instanceid": instanceid,
				}).Error("Unable to gather the state of the instance")

				ticker.Stop()
				return
			}

			//If we're running, then we can actually add the instance to the queue manager
			if state == INSTANCE_STATE_RUNNING {
				// First, let's load our instance
				instance, err := getInstanceByID(instanceid, this.ec2client)
				if err != nil {
					log.WithFields(log.Fields{
						"error":      err.Error(),
						"instanceid": instanceid,
					}).Error("Unable to gather instance information")
				}

				// Build a name for our instance that is relevant
				name := fmt.Sprintf("aws-instance-%s", *instance.PublicIPAddress)

				//Let's actually add and connect to the resource
				resUUID, err := this.q.AddResource(*instance.PublicDNSName, name, this.tls)

				//If there was an error, stop everything and return that we have an error
				if err != nil {
					log.WithFields(log.Fields{
						"error":   err.Error(),
						"address": instance.PublicIPAddress,
						"name":    name,
					}).Error("Unable to connect to AWS resource")

					ticker.Stop()
					return
				}

				// If we successfully connected, then create storage for our local data in the resource manager
				resourceData := resourceInfo{
					Instance:       instance,
					State:          state,
					StartTime:      time.Now(),
					LastUseTime:    time.Now(),
					DisconnectTime: time.Duration(disconnect) * time.Minute,
				}

				// Add it to our local data
				this.resources.Set(resUUID, resourceData)

				// Finally, stop the ticker, because we're done.
				ticker.Stop()
				return
			}
		}
	}
}

/* This function takes the steps necessary to both disconnect the resource from
the queue and then terminate it on AWS.
1. Disconnect the RPC connection gracefully if we can
2. Remove resource from queue
3. Terminate the instance
*/
func (this *awsResourceManager) DeleteResource(resourceid string) error {
	// Remove the resource from the queue
	err := this.q.RemoveResource(resourceid)
	if err != nil {
		return err
	}

	//Let's get the local data in the resource manager
	local, ok := this.resources.Get(resourceid)
	if !ok {
		return errors.New("Unable to gather local AWS resource manager data to delete resource.")
	}
	localresource := local.(resourceInfo)
	ids := []string{
		*localresource.Instance.InstanceID,
	}

	// Now that it's been removed from the queue, we need to terminate it from AWS
	err = termInstance(ids, this.ec2client)

	//No matter what, we want to remove this from the local data because it's missing from the queue
	this.resources.Delete(resourceid)

	// If we had an error terming the instance, then let's remove it.
	if err != nil {
		return err
	}

	return nil
}

func (this awsResourceManager) GetResource(resourceid string) (*queue.Resource, map[string]string, error) {
	resource, ok := this.q.GetResource(resourceid)
	//If we weren't able to gather it, return an error
	if !ok {
		return &queue.Resource{}, nil, errors.New("Resource with requested ID not found in the queue.")
	}

	localdata, ok := this.resources.Get(resourceid)
	if !ok {
		return &queue.Resource{}, nil, errors.New("Could not find local data for resource that was in the queue")
	}
	localres := localdata.(resourceInfo)

	tmpData := make(map[string]string)
	tmpData["instancetype"] = *localres.Instance.InstanceType
	tmpData["disconnect"] = localres.DisconnectTime.String()
	tmpData["subnet"] = *localres.Instance.SubnetID
	tmpData["lastusetime"] = localres.LastUseTime.String()
	tmpData["instanceid"] = *localres.Instance.InstanceID
	tmpData["privateipaddress"] = *localres.Instance.PrivateIPAddress
	tmpData["publicipaddress"] = *localres.Instance.PublicIPAddress
	tmpData["vpcid"] = *localres.Instance.VPCID
	tmpData["instancetype"] = *localres.Instance.InstanceType

	return resource, tmpData, nil
}

func (this *awsResourceManager) UpdateResource(resourceid string, newstatus string, newparams map[string]string) error {
	//Because we need to make some comparisons for pause/resume, let's get the current resource state
	oldresource, _, err := this.GetResource(resourceid)
	if err != nil {
		return err
	}

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

// Get all of the resources managed by this plugin
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

/* Loop through each resource and do the following:
1. Check the AWS status of the resource, if it's terminated let the queue know.
2. See if a resource is in use, if so then update the last used time
3. Check and see if there are any resources that have timed out and terminate them
*/
func (this awsResourceManager) Keep() {
	this.gatherAPIData()

	iter := this.resources.Iterator()
	for data := range iter.Loop() {
		resourceID := data.Key
		resource := data.Val.(resourceInfo)

		//First, let's get the status of the resource
		status, err := getInstanceState(*resource.Instance.VPCID, this.ec2client)

		// Set the state in our local data
		resource.State = status

		if status != INSTANCE_STATE_RUNNING && status != INSTANCE_STATE_PENDING {
			err = this.DeleteResource(resourceID)
			if err != nil {
				log.WithFields(log.Fields{
					"resourceid":      resourceID,
					"resourcemanager": "aws",
					"error":           err.Error(),
				}).Error("Unable to remove stopped instance from the queue.")
				continue
			}
		}

		// 2. Let's check this resource and see if it's being used, if so, set the last use time
		jobs := this.q.AllJobsByResource(resourceID)
		if len(jobs) > 0 {
			resource.LastUseTime = time.Now()
		}

		// 3. Let's check and see if this resource has timed out, if so let's disconnect it
		unusedTime := time.Since(resource.LastUseTime)
		if unusedTime > resource.DisconnectTime {
			err = this.DeleteResource(resourceID)
			if err != nil {
				log.WithFields(log.Fields{
					"resourceid":      resourceID,
					"resourcemanager": "aws",
					"error":           err.Error(),
				}).Error("Unable to remove timed out instance from the queue.")
				continue
			}
		}
	}
}

// This function will gather API data that we'll need for our forms, etc on a regular basis.
// It should be called when the keeper is run, but only once a day or so.
func (this *awsResourceManager) gatherAPIData() {
	log.Debug("Updating AWS API information")

	var err error

	if conf.VPCID != "" {
		this.vpc, err = getVPCByID(conf.VPCID, this.ec2client)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
				"id":    conf.VPCID,
			}).Error("ResMgr (AWS): Unable to gather VPC with ID")
			return
		}
	} else {
		this.vpc, err = getDefaultVPC(this.ec2client)
		if err != nil {
			log.WithField("error", err.Error()).Error("ResMgr (AWS): Unable to gather default VPC.")
			return
		}
	}

	this.subnets, err = getSubnetsInVPC(*this.vpc.VPCID, this.ec2client)
	if err != nil {
		log.WithField("error", err.Error()).Error("ResMgr (AWS): Unable to enumerate subnets in the configured VPC")
		return
	}

	var ok bool
	this.secgrp, ok = getSecurityGroupByName(conf.SecurityGroup, this.ec2client)
	if !ok {
		log.WithField("securitygroup", conf.SecurityGroup).Info("ResMgr (AWS): Unable to find security group, attempting to create it.")
		this.secgrp, err = setupSecurityGroup(conf.SecurityGroup, "Automatically generated by CrackLord AWS ResourceManager", *this.vpc.VPCID, this.ec2client)
		if err != nil {
			log.WithField("error", err.Error()).Error("ResMgr (AWS): Unable to create security group in AWS")
		}
	}
}

// Function to sort the keys of a map and return them
func getSortedKeys(src map[string]string) []string {
	keys := make([]string, len(src))

	i := 0
	for key, _ := range src {
		keys[i] = key
		i++
	}
	sort.Strings(keys)
	return keys
}
