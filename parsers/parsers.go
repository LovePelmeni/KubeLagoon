package parsers

import (
	"encoding/json"
	"sync"
)

// Package consists of the Set of Classes, that Parses Hardware Configuration, User Specified

type Config struct {
	// Represents Configuration of the Virtual Machine
	Mutex sync.RWMutex

	// IP Address of the VM Configuration
	IP struct {
		Hostname string `json:"Hostname" xml:"Hostname"`
		Netmask  string `json:"Netmask" xml:"Netmask"`
		IP       string `json:"IP" xml:"IP"`
		Gateway  string `json:"Gateway" xml:"Gateway"`
	} `json:"IP" xml:"IP"`

	// Hardware Resourcs for the VM Configuration
	Resources struct {
		ResourcePoolUniqueName string `json:"ResourcePoolUniqueName" xml:"ResourcePoolUniqueName"`
		CpuNum                 int32  `json:"CpuNum" xml:"CpuNum"`
		MemoryInMegabytes      int64  `json:"MemoryInMegabytes" xml:"MemoryInMegabytes"`
		ItemPath               string `json:"ItemPath" xml:"ItemPath"`
	} `json:"Resources" xml:"Resources"`

	Disk struct {
		CapacityInKB int `json:"CapacityInKB" xml:"CapacityInKB"`
	} `json:"Disk"`

	// SSH Credentials for the VM
	Ssh struct {
		User     string `json:"User" xml:"User"`
		Password string `json:"Password" xml:"Password"`
	} `json:"Ssh" xml:"Ssh"`

	// Network Resource info, that VM will be Connected to
	Network struct {
		NetworkType       string `json:"NetworkType" xml:"NetworkType"`
		NetworkId         string `json:"NetworkID" xml:"NetworkID"`
		NetworkUniqueName string `json:"NetworkUniqueName" xml:"NetworkUniqueName"`
		ItemPath          string `json:"ItemPath" xml:"ItemPath"`
	} `json:"Network" xml:"Network"`

	// Datacenter Resource Info, VM will be deployed on
	Datacenter struct {
		DatacenterUniqueName string `json:"DatacenterUniqueName" xml:"DatacenterUniqueName"`
		DatacenterName       string `json:"DatacenterName" xml:"DatacenterName"`
		ItemPath             string `json:"ItemPath" xml:"ItemPath"`
	} `json:"Datacenter" xml:"Datacenter"`

	// Datastore Resource Info, VM will be using for storing Data
	DataStore struct {
		DatastoreUniqueName string `json:"DatastoreUniqueName" xml:"DatastoreUniqueName"`
		DatastoreName       string `json:"DataStoreName" xml:"DatastoreName"`
		ItemPath            string `json:"ItemPath" xml:"ItemPath"`
	} `json:"DataStore" xml:"Datastore"`

	// Forder Resource Info, where the Info about VM is going to be Stored.
	Folder struct {
		FolderUniqueName string `json:"FolderUniqueName" xml:"FolderUniqueName"`
		FolderID         string `json:"FolderID" xml:"FolderID"`
		ItemPath         string `json:"ItemPath" xml:"ItemPath"`
	} `json:"Folder" xml:"Folder"`
}

func NewEmptyConfig() *Config {
	return &Config{}
}

type ConfigurationParserInterface interface {
	// Interface, that represents Default Configuration Parser
	// * Parser the Configuration form, and returns set of the Credentials
	// That Will be Potentially used for creating Custom VM...
	ConfigParse(SerializedConfiguraion []byte) (*Config, error)
}

type ConfigurationParser struct {
	ConfigurationParserInterface
	// Main Class, that is used for Parsing the Whole Configuration for the Virtual Server
}

func NewConfigurationParser() *ConfigurationParser {
	return &ConfigurationParser{}
}

func (this *ConfigurationParser) ConfigParse(SerializedConfiguration []byte) (*Config, error) {
	var DecodedConfiguration Config
	JsonDecodeError := json.Unmarshal(SerializedConfiguration, DecodedConfiguration)
	if JsonDecodeError != nil {
		return nil, JsonDecodeError
	} else {
		return &DecodedConfiguration, nil
	}
}
