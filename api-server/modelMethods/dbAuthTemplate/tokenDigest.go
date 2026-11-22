package dbAuthTemplate

import store "api-server/v2/modelMethods/authentication"

func tokenDigest(raw string) string { return store.Digest(raw) }
