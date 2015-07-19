package nmap

import (
	"errors"
	"fmt"
	"os"
	"xml"
)

// Parsing the XML requires a set of structs to match the data we'd like to have.  There are going tobe numerous structs involved in this procexss
type NmapRun struct {
	Info  ScanInfo `xml:"scaninfo>"`
	Hosts []Host   `xml:"host>"`
}

type ScanInfo struct {
	Type        string
	Protocol    string
	NumServices string
	Services    string
}

type Host struct {
	Addresses []Address `xml:"address>"`
	Hostnames []Hostname
	Ports     []Port
}

type Address struct {
	Addr     string
	Addrtype string
}

type Hostname struct {
	Name string
	Type string
}

type Port struct {
	Protocol    string
	PortID      string
	StateInfo   State   `xml:"state>"`
	ServiceInfo Service `xml:"service>"`
}

type State struct {
	State      string
	Reason     string
	Reason_TTL string
}

type Service struct {
	Name   string
	Method string
	Conf   string
}

func parseNmapXML(inputFile string) (NmapRun, error) {
	xmlFile, err := os.Open(inputFile)
	if err != nil {
		return NmapRun{}, err
	}
	defer xmlFile.Close()

	var out NmapRun
	xml.Unmarshal(xmlFile, &out)
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
