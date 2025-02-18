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

## Future features

* Get email gateway working
* Get payment gateway working
* Get a MyBookings page working
