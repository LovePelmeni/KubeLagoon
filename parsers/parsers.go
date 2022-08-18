package parsers

import "sync"

// Package consists of the Set of Classes, that Parses Hardware Configuration, User Specified

type DefaultConfigValueStore struct {
	// Struct, that represents default Values Across all of the Available Fields
	// for every Suggested Object Reference
}

func NewDefaultValueStore() *DefaultConfigValueStore {
	return &DefaultConfigValueStore{}
}

type Config struct {
	// Represents Configuration of the Object Reference
	Mutex sync.RWMutex
	IP    struct {
		Hostname string `json:"Hostname"`
		Netmask  string `json:"Netmask"`
		IP       string `json:"IP"`
	} `json:"IP"`

	Resources struct {
		CpuNum            int64 `json:"CpuNum"`
		MemoryInMegabytes int32 `json:"MemoryInMegabytes"`
	} `json:"Resources"`

	Ssh struct {
		User     string `json:"User"`
		Password string `json:"Password"`
	} `json:"Ssh"`
}

func NewConfig() *Config {
	return &Config{}
}

type ConfigurationParserInterface interface {
	// Interface, that represents Default Configuration Parser
	// * Parser the Configuration form, and returns set of the Credentials
	// That Will be Potentially used for creating Custom VM...
	ConfigParse(Configuration Config) map[string]any
}

type BaseParser struct {
	ConfigurationParserInterface
}

type ConfigurationParser struct {
	ConfigurationParserInterface
	// Main Class, that is used for Parsing the Whole Configuration for the Virtual Server
}

func NewConfigurationParser() *ConfigurationParser {
	return &ConfigurationParser{}
}

func (this *ConfigurationParser) ConfigParse(SerializedConfiguration []byte) (map[string]Config, error) {

	var StructedConfig map[string]string
	ConfigParsers := map[string]ConfigurationParserInterface{
		"Network":      NewNetworkConfigurationParser(),
		"DataStore":    NewDataStoreConfigurationParser(),
		"ResourcePool": NewResourcePoolConfigurationParser(),
		"Folder":       NewFolderConfigurationParser(),
	}

	var ParsedConfigs map[string]Config
	group := sync.WaitGroup{}
	go func() {
		group.Add(1)
		for ConfigName, ConfigValue := range StructedConfig {
			ParsedConfigMap := ConfigParsers[ConfigName].ConfigParse(ConfigValue)
			Config := NewConfig()
			ParsedConfigs[ConfigName] = *Config
		}
		group.Done()
	}()
	return ParsedConfigs, nil
}

func NewBaseParser() *BaseParser {
	return &BaseParser{}
}

func (this *BaseParser) ConfigParse(SerializedConfiguraton string) Config

type NetworkConfigurationParser struct {
	ConfigurationParserInterface
}

func NewNetworkConfigurationParser() *NetworkConfigurationParser {
	return &NetworkConfigurationParser{}
}

func (this *NetworkConfigurationParser) ConfigParse(Configuration Config) map[string]any

type SshConfigurationParser struct {
	ConfigurationParserInterface
}

func NewSshConfigurationParser() *SshConfigurationParser {
	return &SshConfigurationParser{}
}

func (this *SshConfigurationParser) ConfigParse(Configuration Config) map[string]any

type DataCenterConfigurationParser struct {
	ConfigurationParserInterface
}

func NewDatacenterConfigurationParser() *DataCenterConfigurationParser {
	return &DataCenterConfigurationParser{}
}

func (this *DataCenterConfigurationParser) ConfigParse(Configuration Config) map[string]any

type DataStoreConfigurationParser struct {
	ConfigurationParserInterface
}

func NewDataStoreConfigurationParser() *DataStoreConfigurationParser {
	return &DataStoreConfigurationParser{}
}

func (this *DataStoreConfigurationParser) ConfigParse(Configuration Config) map[string]any

type ResourcePoolConfigurationParser struct {
	ConfigurationParserInterface
}

func NewResourcePoolConfigurationParser() *ResourcePoolConfigurationParser {
	return &ResourcePoolConfigurationParser{}
}

func (this *ResourcePoolConfigurationParser) ConfigParse(Configuration Config) map[string]any

type FolderConfigurationParser struct {
	ConfigurationParserInterface
}

func NewFolderConfigurationParser() *FolderConfigurationParser {
	return &FolderConfigurationParser{}
}

func (this *FolderConfigurationParser) ConfigParse(Configuration Config) map[string]any
