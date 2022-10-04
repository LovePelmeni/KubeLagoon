package ssh_rest

import (
	"net/http"
	"os"
	"github.com/LovePelmeni/Infrastructure/authentication"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"github.com/LovePelmeni/Infrastructure/models"
	"github.com/gin-gonic/gin"
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

	// Retrieving Virtual Machine Model Record 

	var VirtualMachine models.VirtualMachine 
	models.Database.Model(&models.VirtualMachine{}).Where(
	"id = ? AND owner_id = ?", VirtualMachineId, VirtualMachineOwnerId).Find(&VirtualMachine)


	// Obtaining Info about the Initializing the SSH Certificates 

	CertificateContent := VirtualMachine.SshInfo.SshPublicKeyMethod.Content 
	CertificateFilename := VirtualMachine.SshInfo.SshPublicKeyMethod.Filename

	// Returning Response
	Context.JSON(http.StatusOK, gin.H{
		"CertificateContent": CertificateContent,
	    "CertificateFilename": CertificateFilename,
	})
} 