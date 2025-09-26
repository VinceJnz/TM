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

I want to create a rust version of the Go wasm client in the folder "client1". Put the rust client into a new folder "client1_rust".
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





## Coding approach
A philosophical question... Should values be passed to methods using the method stuct or should they be passed as function values?

This is a classic design question in Go (and many languages). Here’s a concise, practical answer:

Pass via Method Receiver (Struct Field)
* Use when the value is part of the object’s state (e.g., editor.CurrentRecord).
* Good for values that are persistent or shared across multiple methods.
* Makes sense for "object-oriented" patterns or when the method logically acts on the struct’s data.

Pass as Function Argument
* Use when the value is temporary, context-specific, or not part of the struct’s state.
* Improves testability and reusability (method doesn’t depend on struct state).
* Makes the method’s dependencies explicit.

**Philosophy:**
* Prefer passing as arguments when possible for clarity and testability.
* Use struct fields for persistent state or when the value is truly part of the object.

**In practice:**
* If a value is only needed for a single operation, pass it as an argument.
* If it’s part of the editor’s ongoing state, keep it as a field.

**Summary:**
There’s no single right answer—choose the approach that makes your code clearer, safer, and easier to maintain.



## Docker

<https://learn.microsoft.com/en-us/windows/wsl/wsl-config#configure-global-options-with-wslconfig>

Use the following command steps to compact the docker vhdx (vdisk)

```cmd
wsl --shutdown

docker system prune

diskpart
select vdisk file="C:\Users\Vince2\AppData\Local\Docker\wsl\disk\docker_data.vhdx"
compact vdisk
exit
```


## Connect to a docker image
`docker exec -it <container_name_or_id> /bin/bash`

e.g.
`docker exec -it apiserver-debug /bin/bash`