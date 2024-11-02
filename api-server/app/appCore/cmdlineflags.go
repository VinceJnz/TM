package appCore

import (
	"crypto/tls"
	"flag"
	"fmt"
	"os"
)

func (s *settings) Flags() error {
	//******************************************************************
	// Command line flags and messages
	//******************************************************************
	args := os.Args

	flags := flag.NewFlagSet(args[0], flag.ExitOnError) // args[0] is the program name used at the comand line
	var (
		help         = flags.Bool("help", false, "Optional, prints usage info")
		host         = flags.String("host", "", "Required flag, must be the hostname that is resolvable via DNS, or 'localhost'")
		portHttp     = flags.String("porthttp", "8085", "The http port, defaults to 8085")
		portHttps    = flags.String("porthttps", "8185", "The https port, defaults to 8185")
		apiPrefix    = flags.String("apiprefix", "/api/v1", "The api prefix, defaults to /api/v1")
		serverCert   = flags.String("servercert", "", "Required, the file name of the server's certificate file")
		serverKey    = flags.String("serverkey", "", "Required, the file name of the server's private key file")
		serverCaCert = flags.String("servercacert", "", "Required, the certificate file name of the CA that signed the client’s certificate")
		clientCaCert = flags.String("clientcacert", "", "Required, the certificate file name of the CA that signed the client’s certificate")
		certOpt      = flags.Int("certopt", int(tls.NoClientCert), "Optional, authorization types for authorizing/validating a client’s certificate")
		emailToken   = flags.String("emailtoken", "./certs/gmail/client_token.json", "Optional, the token file name for email server access")
		emailSecret  = flags.String("emailsecret", "./certs/gmail/client_secret.json", "Optional, the secret file for configuration of email access")
		emailAddr    = flags.String("emailaddr", "", "Required, the email address used by the server for sending emails")
		paymentKey   = flags.String("paymentkey", "./certs/stripe/payment_key.json", "Optional, the file for configuration of payment Gateway key")
		//dataSource   = flags.String("datasource", "PM_web_svr_app:123@tcp(localhost:3306)/project_mgnt?parseTime=true", "Optional, the data source name")
		dataSource = flags.String("datasource", "postgres://pm_web_svr_app:123@localhost:5432/project_mgnt?sslmode=disable", "Optional, the data source name")
		logFile    = flags.String("logfile", "", "Optional, the path/name of the log file")
		//dbConfig   = flags.String("dbconfig", "", "Optional, the data build source file")
	)

	if err := flags.Parse(args[1:]); err != nil { //args[1:] is the list of arguments after the program name
		return fmt.Errorf("flags: %w", err)
	}

	usage := `usage:
	
simpleserver -host <hostname> -srvcert <serverCertFile> -cacert <caCertFile> -srvkey <serverPrivateKeyFile> [-port <port> -certopt <certopt> -help]
	
Options:
  -help         Prints this message
  -host         Required, a DNS resolvable host name or 'localhost'.
  -porthttp     Optional, the http port for the server to listen on, defaults to 8085.
  -porthttps    Optional, the https port for the server to listen on, defaults to 8185.
  -srvcert      Required, the name the server's certificate file.
  -srvkey       Required, the name the server's key certificate file.
  -servercacert Required, the certificate file name of the CA that signed the client’s certificate.
  -clientcacert Required, the certificate file name of the CA that signed the client’s certificate.
  -certopt      Optional, authorization types for authorizing/validating a client’s certificate.
                    0=NoClientCert, 1=RequireAnyClientCert, 2=RequireAndVerifyClientCert,
                    3=VerifyClientCertIfGiven, 4=RequireAndVerifyClientCert'.
  -emailtoken   Required, the path and name the json token file for email access.
  -emailsecret  Required, the path and name the json secret files for email access.
  -emailaddr    Required, the email address used by the server for sending emails. 
  -paymentKey   Optional, the file for configuration of payment Gateway key.
  -datasource   Optional, the data source name.
  -logfile      Optional, the path/name of the log file.
  `
	//  -dbconfig     Optional, the name of a sql file used to create the database schema if the schema doesn't exist

	if *help {
		fmt.Println(usage)
		return nil
	}

	s.Host = *host
	s.PortHttp = *portHttp
	s.PortHttps = *portHttps
	s.DataSource = *dataSource
	s.APIprefix = *apiPrefix

	s.CertOpt = *certOpt
	s.ClientCaCert = *clientCaCert
	s.ServerCaCert = *serverCaCert
	s.ServerCert = *serverCert
	s.ServerKey = *serverKey

	s.EmailAddr = *emailAddr
	s.EmailSecret = *emailSecret
	s.EmailToken = *emailToken
	s.PaymentKey = *paymentKey

	s.LogFile = *logFile

	return nil
}
