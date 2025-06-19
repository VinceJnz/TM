package models

import "math/big"

type SRPUser struct {
	ID       int      `json:"id" db:"id"`
	Salt     []byte   `json:"salt" db:"salt"`
	Verifier *big.Int `json:"verifier" db:"verifier"` //[]byte can be converted to/from *big.Int using GobEncode(), GobDecode()
}
type SRPGroup struct {
	GroupID string
}
