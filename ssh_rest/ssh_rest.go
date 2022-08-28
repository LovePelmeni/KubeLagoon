package ssh_rest

import (
	"net/http"
	"log"
	"os"

	"github.com/LovePelmeni/Infrastructure/deploy"
	"github.com/LovePelmeni/Infrastructure/models"
	"github.com/LovePelmeni/Infrastructure/ssh_config"
	"github.com/gin-gonic/gin"
	"github.com/vmware/govmomi/vim25"
	"gorm.io/gorm"
)

var (
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func init() {
	LogFile, Error := os.OpenFile("SshRest.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	DebugLogger = log.New(LogFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLogger = log.New(LogFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(LogFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	if Error != nil {
		panic(Error)
	}
}

func GetCustomerVirtualMachineSSHKeysRestController(RequestContext *gin.Context) {

	var Query []struct {
		gorm.Model
		VirtualMachine models.VirtualMachine // Vm Server
		SSHPublicKey   models.SSHPublicKey   // SSHPublicKey to Access this Server
	} // Represents query of the Joins Pattern

	CustomerId := RequestContext.Query("CustomerId")
	QuerySet := models.Database.Model(&models.VirtualMachine{}).Where("id = ?", CustomerId).Joins(
		"JOIN ssh_public_key ON virtual_machine.id = ssh_public_key.virtual_machine_id").Find(&Query)

	if QuerySet.Error != nil {
		ErrorLogger.Printf(
			"Failed to Obtain the QuerySet of Customer SSH Public Keys, Error: %s", QuerySet.Error)
		RequestContext.JSON(
		http.StatusBadRequest, gin.H{"Error": "Failed to Get Vm SSH Keys"}); return 
	}
	RequestContext.JSON(http.StatusOK, gin.H{"QuerySet": Query})
}

func InitializeVirtualMachineRestController(RequestContext *gin.Context) {
	// Rest Controller, that Initializes SSH Support for the Customer's Virtual Machine Server 
}

func RecoverSSHKeyRestController(RequestContext *gin.Context) {
	// Recovering SSH Keys, by picking them out from the Temp Buffer
}

func UpdateVirtualMachineSshKeysRestController(RequestContext *gin.Context) {
	// Rest Controller, that Allows to Update SSH Key Pairs with new Ones
	VirtualMachineId := RequestContext.Query("VirtualMachineId")
	VmOwnerId := RequestContext.Query("Customerid")

	var SshKeys models.SSHPublicKey
	models.Database.Model(&models.SSHPublicKey{}).Where(
		"virtual_machine_id = ?", VirtualMachineId).Find(&SshKeys)

	VmManager := deploy.NewVirtualMachineManager(vim25.Client{})
	VirtualMachine, FindError := VmManager.GetVirtualMachine(VirtualMachineId, VmOwnerId)

	if FindError != nil {
		RequestContext.JSON(http.StatusBadRequest,
			gin.H{"Error": "Virtual Machine Server not Found"})
		return
	}

	SshManager := ssh_config.NewVirtualMachineSshManager(vim25.Client{}, VirtualMachine)
	PublicKey, PrivateKey, GenerateError := SshManager.GenerateSshKeys()

	if GenerateError != nil {
		RequestContext.JSON(http.StatusBadGateway,
			gin.H{"Error": "Failed to Generate New SSH Keys"})
		return
	}

	switch GenerateError {
	case nil:
		var UpdatedStatus bool = true
		UploadedError := SshManager.UploadSshKeys(*PrivateKey)
		_, Error := SshKeys.Update(PublicKey.Content, PublicKey.FileName)

		if Error != nil {
			UpdatedStatus = false
			ErrorLogger.Printf("Failed to Update SSH keys for the VM wit ID: %s", VirtualMachineId)
		}
		if UploadedError != nil {
			UpdatedStatus = false
			RequestContext.JSON(
				http.StatusCreated, gin.H{"NewPublicKey": PublicKey, "Status": UpdatedStatus})
		}
	default:
		RequestContext.JSON(http.StatusBadGateway,
			gin.H{"Error": "Failed to Generate New SSH Keys"})
	}
}
