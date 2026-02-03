# Build Notes

## Docker build

Docker files can only refer to local files that are in the same folder or sub-folders of the docker file.
To do a multi-stage build for the webserver and the wasm client the dockerfile needed to be able to access
both the client1 and the web-server folders

To do this the docker file needed to be located in the project folder so that the client1 and web-server folders were accssible sub-folders.
This was necessary in the multi-stage build to transfer files:
* wasm.main from wasmbuilder to finalbuilder
* webserver from webbuilder to finalbuilder
* htmel.index from webbuilder to finalbuilder

## VSCode settings

To open a vscode new-window as maximised, change the setting window.newWindowDimensions in Visual Studio Code to "inherit"









## Process for generating key and cert files?

See also: <https://kaushikghosh12.blogspot.com/2016/08/self-signed-certificates-with-microsoft.html>

Install the openssl tool
<https://www.openssl.org/>

or install certstrap (written in golang)

### certstrap

#### Install certstrap

```bat
git clone https://github.com/square/certstrap
cd certstrap
go build
mkdir %UserProfile%\AppData\Local\Programs\tools
copy certstrap.exe %UserProfile%\AppData\Local\Programs\tools\
```

set user path to include `%UserProfile%\AppData\Local\Programs\tools`

#### Using certstrap

refer to the file certs-create.bat

```bat
@ECHO OFF
REM This requires certstrap to be installed (written in golang)
certstrap init --common-name "certauth"
certstrap request-cert --domain "server"
certstrap request-cert --domain "client"
certstrap sign server --CA certauth
certstrap sign client --CA certauth

REM the following line is needed to make the certs available to the browser
cd out
openssl pkcs12 -export -in client.crt -inkey client.key -out client.p12
cd ..
```

### Files
* `.crt` - certificate
* `.csr` - certificate signing request
* `.key` - private key
* `.crl` - certificate revocation list

**Note** The certs have an expiry date. Once the date has past the certs will no longer work and new certs will need to be generated. 

### openssl - Example 1 - Set up CA

First of all, we need a private key `.key` file and a certificate signing request `.csr` file. We are going to use the RSA cryptography to encrypt traffic on our server. To create these files, use the below command.

`openssl req -new -newkey rsa:2048 -nodes -keyout CertName.key -out CertName.csr`

The command above generates `CertName.key` file which is RSA 2048 bits private key file and a CSR in `CertName.csr` file which contains the matching public key.
The command asks for some input information about CSR among which Common Name is critical. This field basically tells the CA about the domain name for which a certificate has to be generated. You can also opt for a wildcard certificate.

Now that we have a CSR, instead of submitting it to a CA, we are going to sign it ourselves using our own private key.

The following is user to generate a new cert or to replace an expired cert.

`openssl x509 -req -days 365 -in CertName.csr -signkey CertName.key -out CertName.crt`

The command above generates a `CertName.crt` certificate file that is self-signed, which also means that it has no root CA.
A browser will not trust a website with this certificate because it does not trust the certificate. To make this work, we need to install this certificate in our system and configure it.
By double-clicking on a certificate file or importing inside the Keychain Access application, you can install a certificate locally. Then you can double click on the certificate to modify its trust parameters.

### openssl - Example 2 - Set up web server (https)

To serve HTTPS content over `http://localhost` domain, the common name of the certificate should be localhost.
The command below generates a certificate signing request (CSR). It generates a `localhost.key` file which is the private key and `localhost.csr` which is the certificate signing request that contains the public key.

`openssl req -new -newkey rsa:2048 -nodes -keyout localhost.key -out localhost.csr`

The next command generates the `localhost.crt` file which is the self-signed certificate signed by our own `localhost.key` private key. The x509 flag states the standard format of an SSL/TLS certificate which is X.509. Since this certificate is not signed by a trusted CA, we need to install it on our system and tweak its trust parameters.

The following is user to generate a new cert or to replace an expired cert.

`openssl x509 -req -days 365 -in localhost.csr -signkey localhost.key -out localhost.crt`

### openssl - Example 3 - Set up wasm client

Repeat the process for the client files
e.g. client.csr, client.key, client.crt

`openssl req -new -newkey rsa:2048 -nodes -keyout client.key -out client.csr`

The following is user to generate a new cert or to replace an expired cert.

`openssl x509 -req -days 365 -in client.csr -signkey client.key -out client.crt`

The following is needed to make the certs available for use in the browser.

`openssl pkcs12 -export -in client.crt -inkey client.key -out client.p12`

### Example 4

<https://certbot.eff.org/instructions?ws=other&os=centosrhel8>

<https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/SSL-on-amazon-linux-2.html#letsencrypt>


Using certbot manually - follow the instructions on the screen

`sudo certbot certonly --manual -d *.CertName.club -d CertName.club`





## SSL Notes

<https://serverfault.com/questions/9708/what-is-a-pem-file-and-how-does-it-differ-from-other-openssl-generated-key-file>

SSL has been around for long enough you'd think that there would be agreed upon container formats. And you're right, there are. Too many standards as it happens. In the end, all of these are different ways to encode Abstract Syntax Notation 1 (ASN.1) formatted data — which happens to be the format x509 certificates are defined in — in machine-readable ways.

* .csr - This is a Certificate Signing Request. Some applications can generate these for submission to certificate-authorities.
 The actual format is PKCS10 which is defined in RFC 2986.
 It includes some/all of the key details of the requested certificate such as subject, organization, state, whatnot, as well as the public key of the certificate to get signed.
 These get signed by the CA and a certificate is returned. The returned certificate is the public certificate (which includes the public key but not the private key), which itself can be in a couple of formats.
* .pem - Defined in RFC 1422 (part of a series from 1421 through 1424) this is a container format that may include just the public certificate
 (such as with Apache installs, and CA certificate files /etc/ssl/certs), or may include an entire certificate chain including public key, private key, and root certificates. 
 Confusingly, it may also encode a CSR (e.g. as used here) as the PKCS10 format can be translated into PEM. 
 The name is from Privacy Enhanced Mail (PEM), a failed method for secure email but the container format it used lives on, and is a base64 translation of the x509 ASN.1 keys.
* .key - This is a (usually) PEM formatted file containing just the private-key of a specific certificate and is merely a conventional name and not a standardized one.
 In Apache installs, this frequently resides in /etc/ssl/private. 
 The rights on these files are very important, and some programs will refuse to load these certificates if they are set wrong.
* .pkcs12 .pfx .p12 - Originally defined by RSA in the Public-Key Cryptography Standards (abbreviated PKCS), the "12" variant was originally enhanced by Microsoft, and later submitted as RFC 7292. 
 This is a password-protected container format that contains both public and private certificate pairs. 
 Unlike .pem files, this container is fully encrypted. Openssl can turn this into a .pem file with both public and private keys: openssl pkcs12 -in file-to-convert.p12 -out converted-file.pem -nodes

A few other formats that show up from time to time:

* .der - A way to encode ASN.1 syntax in binary, a .pem file is just a Base64 encoded .der file. OpenSSL can convert these to .pem (openssl x509 -inform der -in to-convert.der -out converted.pem).
 Windows sees these as Certificate files. By default, Windows will export certificates as .DER formatted files with a different extension. Like...
* .cert .cer .crt - A .pem (or rarely .der) formatted file with a different extension, one that is recognized by Windows Explorer as a certificate, which .pem is not.
* .p7b .keystore - Defined in RFC 2315 as PKCS number 7, this is a format used by Windows for certificate interchange. Java understands these natively, and often uses .keystore as an extension instead.
 Unlike .pem style certificates, this format has a defined way to include certification-path certificates.
* .crl - A certificate revocation list. Certificate Authorities produce these as a way to de-authorize certificates before expiration. You can sometimes download them from CA websites.

In summary, there are four different ways to present certificates and their components:

* PEM - Governed by RFCs, used preferentially by open-source software because it is text-based and therefore less prone to translation/transmission errors. It can have a variety of extensions (.pem, .key, .cer, .cert, more)
* PKCS7 - An open standard used by Java and supported by Windows. Does not contain private key material.
* PKCS12 - A Microsoft private standard that was later defined in an RFC that provides enhanced security versus the plain-text PEM format. This can contain private key and certificate chain material.
 Its used preferentially by Windows systems, and can be freely converted to PEM format through use of openssl.
* DER - The parent format of PEM. It's useful to think of it as a binary version of the base64-encoded PEM file. Not routinely used very much outside of Windows.



## Email notes


## Debugging/Prod

The docker compose file has been reconfigured with profiles to enable debugging

production: `docker compose --profile prod up --build`

debugging: `docker compose --profile debug up --build`


**Debugging workflow:**
* Start containers in debug mode.
* In VS Code, go to the debug tab → select Connect to Docker Delve → press ▶️.
* Set breakpoints in your Go code → execution will pause inside the container.


**Inside the container**
```bash
cd /app
go build -gcflags="all=-N -l" -o apiserver
```

### Run/Stop/Connect demon from terminal
`docker compose --profile prod up --build -d` This builds, then runs the app in demon mode
`docker compose --profile prod up -d` This runs the (already built) app in demon mode
`docker compose --profile prod down`

To connect to the running apiserver via an interactive terminal
`docker exec -it apiserver /bin/sh`