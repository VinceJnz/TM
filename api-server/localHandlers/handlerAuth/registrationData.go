package handlerAuth

import (
	auth "api-server/v2/app/services/authentication"
	"api-server/v2/models"
)

type registrationProfile = auth.RegistrationProfile
type registrationRequest = auth.RegistrationRequest
type pendingRegistration = auth.PendingRegistration

func encodePendingRegistration(r registrationRequest) ([]byte, error) {
	return auth.EncodePendingRegistration(r)
}
func decodePendingRegistration(data string) (models.User, error) {
	return auth.DecodePendingRegistration(data)
}
