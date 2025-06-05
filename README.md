# Trip Manager

An app to allow people to create trips for club members (for a club such as a hiking club).

* Members can book trips
* Admins can set up trips, and pricing
* Sys Admins can change permissions, etc.

## App components

* Rest API server
* PostgreSQL DB server
* WASM client
* Docker config for the build and running of the app

This app is JS free (more or less).

## App trip info

* Name
* Location details
* Owner
* Type
* Dates
* Costs
* Difficulty level
* Max number of participants

## App user info

* Name
* Address
* Email address
* Username
* DOB
* Membership status
* Membership code
* Account status

Note: Passwords are not stored in the database. It uses "Secure Remote Password" <https://pkg.go.dev/github.com/1Password/srp@v0.2.0#section-readme>


## To Do

* Add Rust builder - done
* Create rust wasm code


## Google Jules

I want to create a rust version of the Go wasm clinet in the folder "client1". Put the rust client into a new folder "client1_rust".
Before you start I need to configure the environment.

###  Configuration

```bash
curl --proto '=https' --tlsv1.2 https://sh.rustup.rs -sSf | sh -s -- -y
export PATH="$HOME/.cargo/bin:$PATH"
source "$HOME/.cargo/env"

echo 'export PATH="$HOME/.cargo/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

rustup install 1.82.0
rustup default 1.82.0
rustup toolchain add 1.82.0

cargo install wasm-pack --version 0.12.1

echo 'export PATH="$HOME/.cargo/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

rustc --version
wasm-pack --version
```


```bash
# Install Rust non-interactively
curl --proto '=https' --tlsv1.2 https://sh.rustup.rs -sSf | sh -s -- -y

# Load Rust into current shell session
source "$HOME/.cargo/env"

# Ensure correct Rust version is installed and set as default
rustup install 1.82.0
rustup default 1.82.0

# Install wasm-pack v0.12.1
cargo install wasm-pack --version 0.12.1

# Show installed versions
rustc --version
wasm-pack --version
```



## WebAuthn

Will now use this to authenticate clients.

<https://webauthn.io/>

<https://github.com/go-webauthn/webauthn>

