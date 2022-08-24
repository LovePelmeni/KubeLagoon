package authentication

import (
	"errors"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/LovePelmeni/Infrastructure/models"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

var (
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)
var secretKey = os.Getenv("JWT_SECRET_KEY")

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

	stringToken, error := newToken.SignedString(secretKey)
	if error != nil {
		ErrorLogger.Println("Failed to Stringify JWT Token.")
		return "", error
	}
	return stringToken, nil
}

func ApplyJwtToken(Context gin.Context, Jwt string) gin.Context {

	if Exists, Error := Context.Request.Cookie("jwt-token"); len(Exists.String()) == 0 && Error != nil {
		Cookie, _ := Context.Request.Cookie("jwt-token") // If cookie does still exists
		Cookie.MaxAge = -1                               // it removes the old one, in order to apply the new one.
	}
	Context.SetCookie("jwt-token", Jwt, 10000, "/", "", true, false)
	return Context
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

	DecodedData := &JwtToken{}
	_, Error := jwt.ParseWithClaims(token, DecodedData,
		func(token *jwt.Token) (interface{}, error) { return secretKey, nil })

	if Error != nil {
		InfoLogger.Printf("Invalid Jwt: %s", Error.Error())
		return nil, InvalidJwt()
	}
	return map[string]string{"username": DecodedData.Username,
		"email": DecodedData.Email, "user_id": strconv.Itoa(DecodedData.UserId)}, nil
}