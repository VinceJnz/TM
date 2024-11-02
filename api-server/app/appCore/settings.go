package appCore

import (
	"encoding/json"
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type settings struct {
	Host         string `json:"Host"`
	PortHttp     string `json:"PortHttp"`
	PortHttps    string `json:"PortHttps"`
	DataSource   string `json:"DataSource"`
	APIprefix    string `json:"APIprefix"`
	ServerCaCert string `json:"ServerCaCert"`
	ClientCaCert string `json:"ClientCaCert"`
	ServerKey    string `json:"ServerKey"`
	ServerCert   string `json:"ServerCert"`
	CertOpt      int    `json:"CertOpt"`
	LogFile      string `json:"LogFile"`
	EmailAddr    string `json:"EmailAddr"`
	EmailToken   string `json:"EmailToken"`
	EmailSecret  string `json:"EmailSecret"`
	PaymentKey   string `json:"PaymentKey"`
}

func (s *settings) Save() error {
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

func (s *settings) Load() error {
	var err error
	file, err := os.ReadFile("Project_manager_config.json")
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(file), &s)
	if err != nil {
		return err
	}
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
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	return nil
}

// Validate checks all the json inputs for valid values
func (s *settings) ValidateEnv() error {
	if os.Getenv("HOST") == "" {
		return errors.New("empty setting: HOST missing")
	}
	if os.Getenv("HTTP_PORT") == "" {
		return errors.New("empty setting: HTTP_PORT missing")
	}
	if os.Getenv("HTTPS_PORT") == "" {
		return errors.New("empty setting: HTTPS_PORT missing")
	}
	if os.Getenv("API_PATH_PREFIX") == "" {
		return errors.New("empty setting: API_PATH_PREFIX missing")
	}
	if os.Getenv("DB_USER") == "" {
		return errors.New("empty setting: DB_USER missing")
	}
	if os.Getenv("DB_PASSWORD") == "" {
		return errors.New("empty setting: DB_PASSWORD missing")
	}
	if os.Getenv("DB_NAME") == "" {
		return errors.New("empty setting: DB_NAME missing")
	}
	if os.Getenv("DB_HOST") == "" {
		return errors.New("empty setting: DB_HOST missing")
	}
	if os.Getenv("DB_PORT") == "" {
		return errors.New("empty setting: DB_PORT missing")
	}
	return nil
}
