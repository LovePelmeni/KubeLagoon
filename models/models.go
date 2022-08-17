package models 

type Customer struct {
	Username string  `json:"Username"`
	Email string `json:"Email"`
	Password string `json:"Password"`
	Vms []VirtualMachine 
}

type VirtualMachine struct {
	Host string 
	Port string 
	NetworkIP string 
	SshPublicKey string 
	SshPrivateKey string 
}

