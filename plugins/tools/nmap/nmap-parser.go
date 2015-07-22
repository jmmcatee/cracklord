package nmap

import (
	"encoding/xml"
	"io/ioutil"
	"os"
)

// Parsing the XML requires a set of structs to match the data we'd like to have.  There are going tobe numerous structs involved in this procexss as they represent all of the ways the data could come back from an NMap XML file.
type NmapRun struct {
	Info  ScanInfo `xml:"scaninfo"`
	Hosts []Host   `xml:"host"`
}

type ScanInfo struct {
	Type        string `xml:"type,attr"`
	Protocol    string `xml:"protocol,attr"`
	NumServices string `xml:"numservices,attr"`
	Services    string `xml:"services,attr"`
}

type Host struct {
	Status    string     `xml:"status>state"`
	Addresses []Address  `xml:"address"`
	Hostnames []Hostname `xml:"hostnames"`
	Ports     []Port     `xml:"ports"`
}

type Address struct {
	Addr     string `xml:"addr,attr"`
	Addrtype string `xml:"addrtype,attr"`
}

type Hostname struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
}

type Port struct {
	Protocol    string  `xml:"protocol,attr"`
	PortID      string  `xml:"portid,attr"`
	StateInfo   State   `xml:"state,attr"`
	ServiceInfo Service `xml:"service,attr"`
}

type State struct {
	State      string `xml:"state,attr"`
	Reason     string `xml:"reason,attr"`
	Reason_TTL string `xml:"reason_ttl,attr"`
}

type Service struct {
	Name   string `xml:"name,attr"`
	Method string `xml:"method,attr"`
	Conf   string `xml:"conf,attr"`
}

func parseNmapXML(inputFile string) (NmapRun, error) {
	//Load the XML file that was given as a parameter
	xmlFile, err := os.Open(inputFile)
	if err != nil {
		return NmapRun{}, err
	}
	defer xmlFile.Close()

	//Get all of the data in the XML file
	byteData, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		return NmapRun{}, err
	}

	//Create a struct to hold the data and then unmarshal everything
	var out NmapRun
	err = xml.Unmarshal(byteData, &out)

	return out, err
}

func nmapToCSV(scandata NmapRun) [][]string {
	tmpData := make([][]string, 0)

	for _, host := range scandata.Hosts {
		var ip, ptr string

		//Loop through all the address entires to find the IP
		for _, address := range host.Addresses {
			if address.Addrtype == "ipv4" || address.Addrtype == "ipv6" {
				ip = address.Addr
				break
			}
		}

		//Loop through all the hostname entires to find the PTR record, if there is one
		for _, hostname := range host.Hostnames {
			if hostname.Type == "PTR" {
				ptr = hostname.Name
				break
			}
		}

		if len(host.Ports) <= 0 {
			tmpRow := make([]string, 5)
			tmpRow[0] = ip
			tmpRow[1] = ptr
			tmpRow[2] = "No services/ports up"
			tmpData = append(tmpData, tmpRow)
		}

		//Finally loop through all of the ports and build a row slice of the data
		for _, port := range host.Ports {
			tmpRow := make([]string, 5)
			tmpRow[0] = ip
			tmpRow[1] = ptr
			tmpRow[2] = port.Protocol
			tmpRow[3] = port.PortID
			tmpRow[4] = port.ServiceInfo.Name

			//And then append the data to the master dataset
			tmpData = append(tmpData, tmpRow)
		}
	}

	return tmpData
}
