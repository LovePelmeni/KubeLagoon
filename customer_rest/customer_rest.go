package rest

import (
	"errors"
	"fmt"

	"net/http"
	"os"
	"strconv"

	"reflect"
	"time"

	"github.com/LovePelmeni/Infrastructure/authentication"
	"github.com/LovePelmeni/Infrastructure/models"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	Customer models.Customer
)

var (
	Logger *zap.Logger
)

func InitializeProductionLogger() {

	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)
	file, _ := os.OpenFile("Main.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	logWriter := zapcore.AddSync(file)

	Core := zapcore.NewTee(zapcore.NewCore(fileEncoder, logWriter, zapcore.DebugLevel))
	Logger = zap.New(Core)
}

func init() {
	InitializeProductionLogger()
}

// Authorization Rest API Endpoints

func LoginRestController(RequestContext *gin.Context) {
	// Rest Controller, that is responsible for users to let them login

	Username := RequestContext.PostForm("Username")
	Password := RequestContext.PostForm("Password")

	var Customer models.Customer
	customer := models.Database.Model(&models.Customer{}).Where(
		"username = ?", Username).Find(&Customer)

	if customer.Error != nil {
		RequestContext.JSON(http.StatusBadRequest,
			gin.H{"Error": "No User With This Username Exists :("})
		return
	}

	if EqualsError := bcrypt.CompareHashAndPassword(
		[]byte(Customer.Password), []byte(Password)); EqualsError != nil {
		RequestContext.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid Password"})
		return
	}

	// Generating New Jwt Authentication Token
	NewJwtToken, JwtError := authentication.CreateJwtToken(int(Customer.ID), Customer.Username, Customer.Email)
	if JwtError != nil {
		Logger.Error("Failed to Initialize New Jwt Token", zap.Error(JwtError))
		RequestContext.JSON(http.StatusBadGateway, gin.H{"Error": "Login Error"})
		return
	}

	// Setting UP New Generated Auth Token
	RequestContext.SetCookie("jwt-token", NewJwtToken, int(time.Now().Add(time.Minute*10000).Unix()), "/", "", true, false)
	RequestContext.JSON(http.StatusOK, gin.H{"Status": "Logged In"})
}

func LogoutRestController(RequestContext *gin.Context) {
	// Rest Controller, that is responsible to let users Log out from their existing account

	if Cookie, Error := RequestContext.Cookie("jwt-token"); len(Cookie) != 0 && Error == nil {
		Cookie, _ := RequestContext.Request.Cookie("jwt-token")
		Cookie.Expires.Add(-1)
		RequestContext.JSON(http.StatusOK, gin.H{"Status": "Logged out"})
	}
}

// Customers Rest API Endpoints

func CreateCustomerRestController(RequestContext *gin.Context) {
	// Rest Controller, Responsible for Creating new Customer Profiles

	Username := RequestContext.PostForm("Username")
	Email := RequestContext.PostForm("Email")
	Password := RequestContext.PostForm("Password")
	BillingAddress := RequestContext.PostForm("BillingAddress")
	Country := RequestContext.PostForm("Country")
	ZipCode := RequestContext.PostForm("ZipCode")

	// Checking If Customer is Already Exists...
	var Customer models.Customer
	if Transact := models.Database.Model(
		&models.Customer{}).Where("username = ? OR email = ?",
		Username, Email).Find(&Customer); &Transact.Error == nil || len(Customer.Username) != 0 {
		RequestContext.AbortWithStatusJSON(
			http.StatusBadRequest, gin.H{"Error": "Customer with this Username or Email already exists, Wanna Login?"})
		return
	}

	NewCustomer := models.NewCustomer(Username, Password, Email)
	NewCustomer.BillAddress = BillingAddress
	NewCustomer.Country = Country
	NewCustomer.ZipCode = ZipCode

	Created, Error := NewCustomer.Create()

	if reflect.ValueOf(Created).IsNil() || Error != nil {
		Created.Rollback()

		switch Error {
		case gorm.ErrInvalidData:
			Created.Rollback()
			RequestContext.JSON(http.StatusBadRequest,
				gin.H{"Error": "Invalid Credentials has been Passed, Make sure that Credentials has proper Length and Character Type"})
			return

		case gorm.ErrInvalidValue:
			Created.Rollback()
			RequestContext.JSON(http.StatusBadRequest,
				gin.H{"Error": "Invalid Value has been Passed"})
			return

		case gorm.ErrModelValueRequired:
			Created.Rollback()
			RequestContext.JSON(http.StatusBadRequest,
				gin.H{"Error": "You missed to Setup Required Fields"})
			return

		case gorm.ErrInvalidTransaction:
			Created.Rollback()
			RequestContext.JSON(http.StatusBadRequest,
				gin.H{"Error": "Failed to Perform Transaction"})
			return

		case gorm.ErrPrimaryKeyRequired:
			Created.Rollback()
			RequestContext.JSON(http.StatusBadRequest,
				gin.H{"Error": "Some Fields (`Username` or `Email`) you've specified are already being used"})
			return

		default:
			Created.Rollback()
			RequestContext.JSON(http.StatusBadRequest,
				gin.H{"Error": fmt.Sprintf("Unknown Error `%s`", Error.Error())})
			return
		}
	} else {
		NewJwtToken, JwtError := authentication.CreateJwtToken(
			int(NewCustomer.ID), NewCustomer.Username, NewCustomer.Email)

		if JwtError != nil {
			RequestContext.JSON(http.StatusBadGateway,
				gin.H{"Error": "Failed to Generate Auth Token"})
			return
		}
		Created.Commit()
		RequestContext.SetCookie("jwt-token", NewJwtToken, int(time.Now().Add(10000*time.Minute).Unix()), "/", "", false, false)
		RequestContext.JSON(http.StatusCreated, gin.H{"Operation": "Success"})
	}
}

func ResetPasswordRestController(RequestContext *gin.Context) {
	// Rest Controller, Responsible for Resetting Password

	NewPassword := RequestContext.PostForm("NewPassword")
	CustomerId := RequestContext.PostForm("CustomerId")

	NewPasswordHash, GenerateError := bcrypt.GenerateFromPassword([]byte(NewPassword), 14)
	if GenerateError != nil {
		RequestContext.JSON(http.StatusBadRequest,
			gin.H{"Error": "Failed to Apply new Password"})
	}
	Updated := models.Database.Model(&models.Customer{}).Where(
		"id = ?", CustomerId).Update("Password", NewPasswordHash)

	if Updated.Error != nil {
		Updated.Rollback()
		RequestContext.JSON(
			http.StatusBadGateway, gin.H{"Error": "Oops, Failed to Apply New Password"})
		return
	}

	Updated.Unscoped().Update("Password", NewPasswordHash)
	RequestContext.JSON(http.StatusCreated, gin.H{"Status": "Applied"})

}

func DeleteCustomerRestController(RequestContext *gin.Context) {
	// Rest Controller, Responsible for Deleting Customer Profiles

	token := RequestContext.Request.Header.Get("Authorization")
	Credentials, _ := authentication.GetCustomerJwtCredentials(token)

	Deleted, Error := Customer.Delete(Credentials.UserId)

	switch Error {

	case nil:
		Deleted.Commit()
		RequestContext.JSON(http.StatusCreated, gin.H{"Operation": "Success"})

	case errors.New("%!s(<nil>)"):
		Deleted.Commit()
		RequestContext.JSON(http.StatusCreated, gin.H{"Operation": "Success"})

	case gorm.ErrRecordNotFound:
		Deleted.Rollback()
		RequestContext.JSON(http.StatusBadRequest,
			gin.H{"Error": "Profile with this Credentials Does Not Exist"})

	case gorm.ErrInvalidTransaction:
		Deleted.Rollback()
		Logger.Error(
			"Failed to Delete Customer Profile", zap.String("User ID",
				strconv.Itoa(Credentials.UserId)), zap.Error(Error))
		RequestContext.JSON(http.StatusBadGateway,
			gin.H{"Error": "Failed to Delete Profile"})

	default:
		Deleted.Rollback()
		Logger.Error("Unknown Error", zap.Error(Error))
		RequestContext.JSON(http.StatusBadGateway,
			gin.H{"Error": fmt.Sprint("Failed to Delete Profile, Please Contact Support")})
	}
}

func GetCustomerProfileRestController(RequestContext *gin.Context) {
	// Returns Customer's Profile, based on the Jwt token passed
	Token := RequestContext.GetHeader("Authorization")
	if len(Token) == 0 {
		RequestContext.JSON(http.StatusForbidden, gin.H{"Error": "UnAuthorized"})
		return
	}
	JwtCredentials, JwtError := authentication.GetCustomerJwtCredentials(Token)
	if JwtError != nil {
		RequestContext.JSON(http.StatusForbidden, gin.H{"Error": "Invalid Jwt Token"})
		return
	}

	if Customer := models.Database.Model(&models.Customer{}).Where("id = ?", JwtCredentials.UserId,
		JwtCredentials.Username, JwtCredentials.Email).Find(&Customer); Customer.Error != nil {
		RequestContext.JSON(
			http.StatusBadRequest, gin.H{"Error": "No Such Profile has been Found"})
		return
	} else {
		RequestContext.JSON(http.StatusOK, gin.H{"Profile": Customer})
		return
	}
}

func SupportRestController(RequestContext *gin.Context) {
	// Rest Controller, that is Responsible for Sending out Messages / Notifications to the Support Email
}
