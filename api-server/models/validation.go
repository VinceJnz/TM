package models

import (
	"api-server/v2/internal/fieldrules"
	"reflect"
)

// ValidateFieldRules rejects programmer errors before accepting API traffic.
func ValidateFieldRules() error {
	for _, v := range []any{AccountWrite{}, ProfileWrite{}, EmailChangeRequest{}, EmailChangeProof{}, GroupWrite{}, GroupResourceWrite{}, UserGroupWrite{}, TripCostWrite{}, TripCostGroupWrite{}, BookingWrite{}, TripWrite{}} {
		if err := fieldrules.ValidateType(reflect.TypeOf(v)); err != nil {
			return err
		}
	}
	return nil
}
