package main

import (
	"api-server/v2/app/authorization"
	"strings"
)

func applyWriteRules(name string, schema, properties map[string]any) {
	required := map[string][]string{
		"GroupResourceWrite": {"group_id", "resource_id", "access_level_id", "access_scope_id"},
		"UserGroupWrite":     {"user_id", "group_id"},
		"TripCostWrite":      {"trip_cost_group_id", "member_status_id", "user_age_group_id", "season_id"},
		"TripCostGroupWrite": {"description"},
	}
	if fields := required[name]; len(fields) > 0 {
		schema["required"] = fields
	}
	for _, field := range required[name] {
		p := properties[field].(map[string]any)
		if strings.HasSuffix(field, "_id") {
			p["minimum"] = 1
		}
	}
	if name == "TripCostGroupWrite" {
		p := properties["description"].(map[string]any)
		p["minLength"] = 1
		p["maxLength"] = 50
		p["description"] = "Trimmed, nonempty description"
	}
	if p, ok := properties["amount"].(map[string]any); ok {
		p["minimum"] = 0
		p["maximum"] = 99999999.99
		p["multipleOf"] = 0.01
	}
}
func decorateOperations(paths map[string]any) {
	for path, value := range paths {
		resource := strings.Split(strings.TrimPrefix(path, "/"), "/")[0]
		item := value.(map[string]any)
		for method, operation := range item {
			if method == "parameters" {
				continue
			}
			op := operation.(map[string]any)
			op["x-permission-resource"] = resource
			scopes := []string{"any"}
			if authorization.SupportsScope(resource, strings.ToUpper(method), "own") {
				scopes = append(scopes, "own")
			}
			op["x-supported-scopes"] = scopes
			responses := op["responses"].(map[string]any)
			for _, code := range []string{"400", "401", "403", "404", "409", "422", "503"} {
				if responses[code] == nil {
					responses[code] = map[string]any{"description": "Request validation, authorization, record conflict or dependency failure", "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "object", "properties": map[string]any{"error": map[string]any{"type": "string"}, "message": map[string]any{"type": "string"}}}}}}
				}
			}
			for _, code := range []string{"400", "401", "403", "404", "409", "422", "503"} {
				response := responses[code].(map[string]any)
				if content, ok := response["content"].(map[string]any); ok {
					content["text/plain"] = map[string]any{"schema": map[string]any{"type": "string"}}
				}
			}
		}
	}
}
