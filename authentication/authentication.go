package authentication

import (
	"errors"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Logger *zap.Logger
)
var secretKey = []byte(os.Getenv("JWT_SECRET_KEY"))

func InitializeProductionLogger() {

	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)
	file, _ := os.OpenFile("AuthenticationLog.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	logWriter := zapcore.AddSync(file)

	Core := zapcore.NewTee(zapcore.NewCore(fileEncoder, logWriter, zapcore.DebugLevel))
	Logger = zap.New(Core)
}

func init() {
	InitializeProductionLogger()
}

// Packege, that Is Responsible for handling Customer Authentication Policy

func InvalidJwt() error {
	return errors.New("Invalid Jwt Token has been Passed")
}

type JwtToken struct {
	jwt.StandardClaims

	UserId   int
	Username string
	Email    string
}

func CreateJwtToken(UserId int, Username string, Email string) (string, error) {

	newToken := jwt.New(jwt.SigningMethodHS256)
	newTokenClaims := newToken.Claims.(jwt.MapClaims)

	newTokenClaims["UserId"] = UserId
	newTokenClaims["Username"] = Username
	newTokenClaims["Email"] = Email
	newTokenClaims["exp"] = time.Now().Add(10000 * time.Minute).Unix()
	stringToken, Error := newToken.SignedString(secretKey)
	if Error != nil {
		Logger.Error("Failed to Stringify JWT Token. Error: %s", zap.Error(Error))
		return "", Error
	}
	return stringToken, nil
}

type JwtValidator struct {
	Token string
}

type DecodedJwtData struct {
	UserId   int    `json:"UserId"`
	Username string `json:"Username"`
	Email    string `json:"Email"`
}

func CheckValidJwtToken(token string) error {

	// Checks if the Customer's jwt auth Token is Valid.

	DecodedData := &JwtToken{}
	_, Error := jwt.ParseWithClaims(token, DecodedData,
		func(token *jwt.Token) (interface{}, error) { return secretKey, nil })

	if Error != nil {
		return InvalidJwt()
	}
	return nil
}

func GetCustomerJwtCredentials(token string) (*JwtToken, error) {
	// Returns Decoded Customer Credentials from the Jwt Auth Token

	if len(token) == 0 {
		return nil, errors.New("Invalid Jwt Token")
	}
	DecodedData := &JwtToken{}
	_, Error := jwt.ParseWithClaims(token, DecodedData,
		func(token *jwt.Token) (interface{}, error) { return secretKey, nil })

	if Error != nil {
		return nil, Error
	}
	return DecodedData, nil
}