package helpers

import "api-server/v2/models"

// RedactUserForClient returns a copy of the user with sensitive fields cleared
// before serializing responses back to API clients.
func RedactUserForClient(user models.User) models.User {
	redacted := user
	redacted.Password = models.User{}.Password
	redacted.ProviderID = models.User{}.ProviderID
	return redacted
}

// RedactUserForPublicProfile applies a stricter redaction policy suitable for
// public-profile style responses.
func RedactUserForPublicProfile(user models.User) models.User {
	redacted := RedactUserForClient(user)
	redacted.Address = models.User{}.Address
	redacted.BirthDate = models.User{}.BirthDate
	redacted.MemberCode = models.User{}.MemberCode
	return redacted
}
