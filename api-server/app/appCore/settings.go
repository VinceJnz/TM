package appCore

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type settings struct {
	AppTitle           string `json:"AppTitle"`
	Host               string `json:"Host"`
	PortHttp           string `json:"PortHttp"`
	PortHttps          string `json:"PortHttps"`
	DataSource         string `json:"DataSource"`
	APIprefix          string `json:"APIprefix"`
	ServerCaCert       string `json:"ServerCaCert"`
	ClientCaCert       string `json:"ClientCaCert"`
	ServerKey          string `json:"ServerKey"`
	ServerCert         string `json:"ServerCert"`
	CertOpt            int    `json:"CertOpt"`
	LogFile            string `json:"LogFile"`
	EmailAddr          string `json:"EmailAddr"`
	EmailToken         string `json:"EmailToken"`
	EmailSecret        string `json:"EmailSecret"`
	PaymentKey         string `json:"PaymentKey"`
	GoogleClientID     string `json:"GoogleClientID"`
	GoogleClientSecret string `json:"GoogleClientSecret"`
	GoogleRedirectURL  string `json:"GoogleRedirectURL"`
	SessionKey         string `json:"SessionKey"`
	ClientRedirectURL  string `json:"ClientRedirectURL"`
	EmailDebugAddr     string `json:"EmailDebugAddr"`
	DevMode            bool   `json:"DevMode"`
}

func (s *settings) SaveJson() error {
	var err error
	file, err := json.MarshalIndent(s, "", "	")
	if err != nil {
		return err
	}
	err = os.WriteFile("Project_manager_config.json", file, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (s *settings) LoadJson() error {
	var err error
	envData, err := os.ReadFile("Project_manager_config.json")
	if err != nil {
		log.Printf("%sload() Error loading settings file: %v\n", debugTag, err)
		return err
	}
	log.Printf("%sload()1 Settings file loaded successfully: %s\n", debugTag, string(envData))
	err = json.Unmarshal(envData, &s)
	if err != nil {
		log.Printf("%sload()1 Error loading settings: %v\n", debugTag, err)
		return err
	}
	log.Printf("%sload()3 Settings loaded successfully: %+v\n", debugTag, s)
	return nil
}

// Validate checks all the json inputs for valid values
func (s *settings) Validate() error {
	if s.Host == "" {
		return errors.New("empty setting: Host name missing")
	}
	if s.PortHttp == "" {
		return errors.New("empty setting: PortHttp name missing")
	}
	if s.PortHttps == "" {
		return errors.New("empty setting: PortHttps name missing")
	}
	if s.DataSource == "" {
		return errors.New("empty setting: DataSource name missing")
	}
	if s.APIprefix == "" {
		return errors.New("empty setting: APIprefix name missing")
	}
	return nil
}

func (s *settings) LoadEnv() error {
	var err error
	err = godotenv.Load() //default is to load .env file in the current directory
	if err != nil {
		return fmt.Errorf("error loading .env file (required): %w", err)
	}
	s.Host = os.Getenv("HOST")
	s.PortHttp = os.Getenv("HTTP_PORT")
	s.PortHttps = os.Getenv("HTTPS_PORT")
	s.APIprefix = os.Getenv("API_PATH_PREFIX")
	s.DataSource = os.Getenv("DATA_SOURCE")
	if s.DataSource == "" {
		dbHost := os.Getenv("DB_HOST")
		dbPort := os.Getenv("DB_PORT")
		dbUser := os.Getenv("DB_USER")
		dbPassword := os.Getenv("DB_PASSWORD")
		dbName := os.Getenv("DB_NAME")
		if dbHost != "" && dbPort != "" && dbUser != "" && dbPassword != "" && dbName != "" {
			s.DataSource = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", dbHost, dbPort, dbUser, dbPassword, dbName)
		}
	}
	s.ServerCaCert = os.Getenv("SERVER_CA_CERT")
	s.ClientCaCert = os.Getenv("CLIENT_CA_CERT")
	s.ServerKey = os.Getenv("SERVER_KEY")
	s.ServerCert = os.Getenv("SERVER_CERT")
	s.CertOpt, err = strconv.Atoi(os.Getenv("CERT_OPTION"))
	if err != nil {
		s.CertOpt = 0 // Default value if conversion fails
		log.Printf("%sLoadEnv() Warning: converting CERT_OPTION to int: %v, supplied value is not an integer, using default value 0\n", debugTag, err)
	}
	s.LogFile = os.Getenv("LOG_FILE")
	s.EmailAddr = os.Getenv("EMAIL_ADDR")
	s.EmailToken = os.Getenv("EMAIL_TOKEN")
	s.EmailSecret = os.Getenv("EMAIL_SECRET")
	s.PaymentKey = os.Getenv("PAYMENT_KEY")
	s.GoogleClientID = os.Getenv("GOOGLE_CLIENT_ID")
	s.GoogleClientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	s.GoogleRedirectURL = os.Getenv("GOOGLE_REDIRECT_URL")
	s.SessionKey = os.Getenv("SESSION_KEY")
	s.ClientRedirectURL = os.Getenv("CLIENT_REDIRECT_URL")
	s.EmailDebugAddr = os.Getenv("EMAIL_DEBUG_ADDR")
	s.DevMode = os.Getenv("DEV_MODE") == "true"
	s.AppTitle = os.Getenv("APP_TITLE")
	if err := s.ValidateEnv(); err != nil {
		return err
	}
	return nil
}

// Validate checks all the json inputs for valid values
func (s *settings) ValidateEnv() error {
	if s.Host == "" {
		log.Printf("%sValidateEnv() Warning: HOST is not set, using default value\n", debugTag)
		return errors.New("empty setting: HOST missing")
	}
	if s.PortHttp == "" {
		log.Printf("%sValidateEnv() Warning: HTTP_PORT is not set, using default value\n", debugTag)
		return errors.New("empty setting: HTTP_PORT missing")
	}
	if s.PortHttps == "" {
		log.Printf("%sValidateEnv() Warning: HTTPS_PORT is not set, using default value\n", debugTag)
		return errors.New("empty setting: HTTPS_PORT missing")
	}
	if s.APIprefix == "" {
		log.Printf("%sValidateEnv() Warning: API_PATH_PREFIX is not set, using default value\n", debugTag)
		return errors.New("empty setting: API_PATH_PREFIX missing")
	}
	if s.DataSource == "" {
		log.Printf("%sValidateEnv() Warning: DATA_SOURCE (or DB_* fallback) is not set, using default value\n", debugTag)
		return errors.New("empty setting: DATA_SOURCE missing")
	}
	if s.AppTitle == "" {
		log.Printf("%sValidateEnv() Warning: APP_TITLE is not set, using default value\n", debugTag)
		return errors.New("empty setting: APP_TITLE missing")
	}
	if s.EmailDebugAddr != "" {
		log.Printf("%sValidateEnv() Warning: EMAIL_DEBUG_ADDR is set, using DEBUG value\n", debugTag)
		return errors.New("nonempty setting: EMAIL_DEBUG_ADDR should be empty in production")
	}
	return nil
}
