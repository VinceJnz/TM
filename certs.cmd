@ECHO OFF
REM This requires certstrap to be installed (written in golang)

REM Save current directory
SET OLDDIR=%CD%

REM Change to certs folder
CD /D "%~dp0certs"
REM CD /D ".\certs"

REM Create directories if they don't exist
IF NOT EXIST ".\api-server" mkdir "\api-server"
IF NOT EXIST ".\cert-auth" mkdir "\cert-auth"
IF NOT EXIST ".\client" mkdir "\client"
IF NOT EXIST ".\out" mkdir "\out"

REM Generate CA and certs
certstrap init --common-name "TM_test" REM ```--expires "730"``` REM CA cert -- 2 years (730 days)
certstrap request-cert --domain "localhost" REM ```--expires "730"``` REM cert for the server -- 2 years (730 days)
certstrap request-cert --domain "client" REM ```--expires "730"``` REM cert for the client -- 2 years (730 days)
certstrap sign localhost --CA TM_test
certstrap sign client --CA TM_test

REM Move CA certs to cert-auth
move .\out\TM_test.crt .\cert-auth\certauth.crt
move .\out\TM_test.key .\cert-auth\certauth.key

REM Move server certs to api-server
move .\out\localhost.crt .\api-server\localhost.crt
move .\out\localhost.key .\api-server\localhost.key

REM Move client certs to client
move .\out\client.crt .\client\client.crt
move .\out\client.key .\client\client.key

REM the following line is needed to make the certs available to the browser
openssl pkcs12 -export -in .\client\client.crt -inkey .\client\client.key -out .\client\client.p12

REM filepath: d:\Users\Vince\Documents\GitHub\TM\certs\update-certs.cmd

REM Paths to certs
set CACERT=.\cert-auth\certauth.crt
set CLIENTCERT=.\client\client.crt
set CLIENTKEY=.\client\client.key
set OUTFILE=.\api-server\certs.go

REM Write Go file header
echo // Auto-generated certs.go > %OUTFILE%
echo package certs >> %OUTFILE%
echo. >> %OUTFILE%

REM Add CA cert
echo // CA Certificate (.crt) >> %OUTFILE%
echo var CaCert = []byte(` >> %OUTFILE%
type %CACERT% >> %OUTFILE%
echo `) >> %OUTFILE%
echo. >> %OUTFILE%

REM Add client cert
echo // Client Certificate (.crt) >> %OUTFILE%
echo var ClientCert = []byte(` >> %OUTFILE%
type %CLIENTCERT% >> %OUTFILE%
echo `) >> %OUTFILE%
echo. >> %OUTFILE%

REM Add client key
echo // Client Key (.key) >> %OUTFILE%
echo var ClientKey = []byte(` >> %OUTFILE%
type %CLIENTKEY% >> %OUTFILE%
echo `) >> %OUTFILE%

REM Change back to original directory
CD /D "%OLDDIR%"
