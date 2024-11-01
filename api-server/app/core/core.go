package core

import (
	"api-server/v2/app/dataStore"
	"encoding/json"
	"errors"
	"os"

	"github.com/gorilla/mux"
)

//const debugTag = "core."

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

// App - Stores the application configuration and status
type Config struct {
	Mux *mux.Router
	Db  *dataStore.DB
	//Service *store.Service
	//EmailSvc *gmailGW.Handler
	//PaymentSvc *stripeGW.Handler
	Settings settings
}

// NewApp creates a new App and Initiliases the DB connection
func New() *Config {
	//err = a.Settings.Load()
	//if err != nil {
	//	log.Fatalf("%v", debugTag+"New()1 - settings file (Project_manager_config.json) not found")
	//	return err
	//}

	return &Config{}
}

func (a *Config) Run() error {
	var err error

	//a.Db, err = dataStore.NewMysql(a.Settings.DataSource)
	a.Db, err = dataStore.NewPostgres(a.Settings.DataSource)
	if err != nil {
		return err
	}

	//app, err = application.CreateDbSchema(*dataSource, *dbConfig)
	//if err != nil {
	//	log.Fatal(err)
	//	return fmt.Errorf("createDbSchema(): %w", err)
	//}

	//a.Service = store.New(a.Db)
	a.Mux = mux.NewRouter()

	//a.EmailSvc = gmailGW.New(a.Settings.EmailSecret, a.Settings.EmailToken, a.Settings.EmailAddr)
	//a.PaymentSvc = stripeGW.New(a.Db, a.WsPool, a.Settings.PaymentKey, "https://"+a.Settings.Host+":"+a.Settings.PortHttps+a.Settings.APIprefix)
	//a.PaymentSvc = stripeGW.New(a.Db, a.Settings.PaymentKey, "https://"+a.Settings.Host+":"+a.Settings.PortHttps+a.Settings.APIprefix)

	//a.WsPool = tmWebsocket.NewPool()
	//a.WsClientPool = ctrlWebsocketGorilla.NewHub()
	//go a.WsClientPool.Run()
	return nil
}

func (a *Config) Close() { //Should this be a pointer ???????????
	a.Db.Close()
}

// Add a config file reader ********************* something like the following:
/*
// config the settings variable
var config = &configuration{}

// configuration contains the application settings
type configuration struct {
	Database  database.Info   `json:"Database"`
	Email     email.SMTPInfo  `json:"Email"`
	Recaptcha recaptcha.Info  `json:"Recaptcha"`
	Server    server.Server   `json:"Server"`
	Session   session.Session `json:"Session"`
	Template  view.Template   `json:"Template"`
	View      view.View       `json:"View"`
}

// ParseJSON unmarshals bytes to structs
func (c *configuration) ParseJSON(b []byte) error {
	return json.Unmarshal(b, &c)
}
*/
