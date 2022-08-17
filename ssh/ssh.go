package ssh_manager

import (
	"github.com/vmware/govmomi/object"
)
func init() {
}
type SshCredentials struct {
	User       string `json:"User"`
	PublicKey  string `json:"PublicKey"`
	PrivateKey string `json:"PrivateKey"`
}

func NewSshCredentials(PublicKey string) *SshCredentials {
	return &SshCredentials{}
}

type VirtualMachineSshManagerInterface interface {
}

type VirtualMachineSshManager struct {
}

func NewVirtualMachineSshManager() *VirtualMachineSshManager{
	return &VirtualMachineSshManager{}
}

func (this *VirtualMachineSshManager) SetupSsh(
VirtualMachine *object.VirtualMachine) (SshCredentials, error)