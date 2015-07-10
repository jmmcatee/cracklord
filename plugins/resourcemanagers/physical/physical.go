package physical

import ()

type physicalResourceManager struct {
}

func (p physicalResourceManager) SystemName() string {

}

func (p physicalResourceManager) DisplayName() string {

}

func (p physicalResourceManager) Description() string {

}

func (p physicalResourceManager) Parameters() string {

}

func (p *physicalResourceManager) AddResource(name string, address string, params map[string]string, tls *tls.Config) error {

}

func (p *physicalResourceManager) DeleteResource(resourceid string) error {

}

func (p *physicalResourceManager) GetResource(resourceid string) (Resource, error) {

}

func (p *physicalResourceManager) PauseResource(resourceid string) error {

}

func (p *physicalResourceManager) ResumeResource(resourceid string) error {

}

func (p *physicalResourceManager) GetManagedResources() []string {

}

func (p *physicalResourceManager) Keep() {

}
