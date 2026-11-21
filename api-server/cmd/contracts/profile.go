package main

func addProfilePaths(paths map[string]any) {
	content := func(name string) map[string]any {
		return map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/" + name}}}
	}
	write := func(input, status, description string) map[string]any {
		return map[string]any{"requestBody": map[string]any{"required": true, "content": content(input)}, "responses": map[string]any{status: map[string]any{"description": description}}}
	}
	paths["/myProfile"] = map[string]any{
		"get": map[string]any{"responses": map[string]any{"200": map[string]any{"description": "The session user's profile; a single object", "content": content("ProfileDTO")}}},
		"put": write("ProfileWrite", "200", "Profile saved. The email field must match the current address; email changes use the separate verification flow."),
	}
	request := write("EmailChangeRequest", "202", "Approval codes sent to the current and proposed inboxes. Email is unchanged until both proofs are confirmed.")
	request["responses"].(map[string]any)["429"] = map[string]any{"description": "Wait at least 30 seconds before requesting replacement codes"}
	paths["/myProfile/email-change"] = map[string]any{"post": request}
	paths["/myProfile/email-change/confirm"] = map[string]any{"post": write("EmailChangeProof", "200", "Email changed; all existing sessions are revoked. Sign in again. Proofs expire after 15 minutes and five failed attempts.")}
}
