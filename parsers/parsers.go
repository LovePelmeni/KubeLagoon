package parsers 

type ConfigurationParserInterface interface {
	// Interface, that represents Default Configuration Parser 
	// * Parser the Configuration form, and returns set of the Credentials 
	// That Will be Potentially used for creating Custom VM...
	ConfigParse() any 
}

type PersistentDiskConfigurationParser struct {

}

type SshConfigurationParser struct {
	
}
