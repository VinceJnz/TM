package handlerAuth

import "api-server/v2/app/authorization"

func supportsScope(resource, method, scope string) bool {
	return authorization.SupportsScope(resource, method, scope)
}
