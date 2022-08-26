package authentication

import (
	"errors"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/LovePelmeni/Infrastructure/models"
	"github.com/dgrijalva/jwt-go"
)

var (
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)
var secretKey = os.Getenv("JWT_SECRET_KEY")

func init() {
	LogFile, LogError := os.OpenFile("Authentication.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if LogError != nil {
		panic(LogError)
	}
	DebugLogger = log.New(LogFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLogger = log.New(LogFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(LogFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
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
	stringToken, Error := newToken.SignedString(string(secretKey))
	if Error != nil {
		ErrorLogger.Printf("Failed to Stringify JWT Token. Error: %s", Error)
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

	DecodedData := &JwtToken{}
	_, Error := jwt.ParseWithClaims(token, DecodedData,
		func(token *jwt.Token) (interface{}, error) { return secretKey, nil })

	if Error != nil {
		InfoLogger.Printf("Jwt Error: %s", Error.Error())
		return InvalidJwt()
	}

	if customer := models.Database.Table("customers").Where("username = ? AND email = ?",
		DecodedData.Username, DecodedData.Email); customer.Error != nil {
		return InvalidJwt()
	}
	return nil
}

func GetCustomerJwtCredentials(token string) (map[string]string, error) {

	if len(token) == 0 {
		return nil, errors.New("Invalid Jwt Token")
	}
	DecodedData := &JwtToken{}
	_, Error := jwt.ParseWithClaims(token, DecodedData,
		func(token *jwt.Token) (interface{}, error) { return secretKey, nil })

	if Error != nil {
		return nil, Error
	}
	return map[string]string{"username": DecodedData.Username,
		"email": DecodedData.Email, "user_id": strconv.Itoa(DecodedData.UserId)}, nil
}
