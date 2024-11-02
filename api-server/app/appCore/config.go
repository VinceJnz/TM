package appCore

import (
	"encoding/json"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
)

type ContextKey string

type Config struct {
	Db        *sqlx.DB
	UserIDKey ContextKey
	Settings  settings
}

func IdKey() ContextKey {
	return ContextKey("userID")
}

func New() *Config {
	db, err := InitDB()
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	userIDKey := IdKey()
	return &Config{
		Db:        db,
		UserIDKey: userIDKey,
	}
}

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
