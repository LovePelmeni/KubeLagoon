package ssh_rest

import (
	"net/http"

	"os"

	"github.com/LovePelmeni/Infrastructure/authentication"
	"github.com/LovePelmeni/Infrastructure/deploy"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/LovePelmeni/Infrastructure/models"
	"github.com/LovePelmeni/Infrastructure/ssh_config"

	"github.com/gin-gonic/gin"
	"github.com/vmware/govmomi/vim25"
)

var (
	Logger *zap.Logger
)

func InitializeProductionLogger() {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)
	file, _ := os.OpenFile("SshRestLog.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	logWriter := zapcore.AddSync(file)

	Core := zapcore.NewTee(zapcore.NewCore(fileEncoder, logWriter, zapcore.DebugLevel))
	Logger = zap.New(Core)
}

func init() {
	// Initializing Logger 
	InitializeProductionLogger()
}

func GetDownloadPublicSshCertificateRestController(Context *gin.Context) {
	// Rest Controller, that returns Download File for the Virtual Machine Server Ssh Option 

	JwtCustomerCredentials, _ := authentication.GetCustomerJwtCredentials(Context.GetHeader("Authorization"))
	VirtualMachineId := Context.Query("VirtualMachineId")
	VirtualMachineOwnerId := JwtCustomerCredentials.Id
	Client := vim25.Client{}

	SshManager := ssh_config.NewVirtualMachineSshCertificateManager(Client)
	VmManager := deploy.NewVirtualMachineManager(Client)

	// Retrieving Virtual Machine Instance 
	VirtualMachineInstance, FindError := VmManager.GetVirtualMachine(VirtualMachineId, VirtualMachineOwnerId)
	if FindError != nil {Logger.Debug("Failed to Retrieve Virtual Machine", zap.Error(FindError))}

	// Retrieving Virtual Machine Model Record 

	var VirtualMachine models.VirtualMachine 
	models.Database.Select("SshInfo").Model(&models.VirtualMachine{}).Where(
	"id = ? AND owner_id = ?", VirtualMachineId, VirtualMachineOwnerId).Find(&VirtualMachine)


	// Retrieving Certificate Public Ssh Key
	SshPublicCertificateContent, CertificateError := SshManager.GetSshPublicCertificate(
	VirtualMachineInstance, VirtualMachine.SshInfo.SshCredentialsInfo)

	if CertificateError != nil {Logger.Error("Certificate Error"); Context.JSON(http.StatusBadGateway,
    gin.H{"Error": "Failed to Retrieve Certificate"}); return}

	// Returning Response
	Context.JSON(http.StatusOK, gin.H{"CertificateContent": SshPublicCertificateContent})
} 