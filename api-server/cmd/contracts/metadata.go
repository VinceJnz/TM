package main

import (
	"api-server/v2/app/authorization"
	"api-server/v2/internal/fieldrules"
	"api-server/v2/models"
	"encoding/json"
	"reflect"
	"strings"
)

func applyWriteRules(t reflect.Type, schema, properties map[string]any) {
	required := []string{}
	for _, rule := range fieldrules.Fields(t) {
		if rule.Required {
			required = append(required, rule.Name)
		}
		property := properties[rule.Name].(map[string]any)
		for name, value := range rule.Constraints {
			property[name] = value
		}
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Type != reflect.TypeOf(models.Money{}) {
			continue
		}
		property := properties[strings.Split(field.Tag.Get("json"), ",")[0]].(map[string]any)
		property["minimum"] = 0
		property["maximum"] = json.Number(models.MoneyMaximum)
		property["multipleOf"] = json.Number("0.01")
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
			if resource == "users" && (method == "put" || method == "delete") {
				op["parameters"] = []any{map[string]any{"name": "X-Change-Reason", "in": "header", "required": method == "delete", "description": "Required for deletion and changes to email, username or account status; revokes the affected user's tokens.", "schema": map[string]any{"type": "string", "minLength": 1, "maxLength": 500}}}
			}
			if method == "get" && !strings.Contains(path, "{id}") && (resource == "securityUserGroup" || resource == "securityGroupResource") {
				field := "user_id"
				if resource == "securityGroupResource" {
					field = "group_id"
				}
				parameters, _ := op["parameters"].([]any)
				op["parameters"] = append(parameters, map[string]any{"name": field, "in": "query", "description": "Parent filter applied before pagination", "schema": map[string]any{"type": "integer", "minimum": 1}})
			}
			op["x-permission-resource"] = resource
			scopes := []string{"any"}
			if authorization.SupportsScope(resource, strings.ToUpper(method), "own") {
				scopes = append(scopes, "own")
			}
			op["x-supported-scopes"] = scopes
			responses := op["responses"].(map[string]any)
			for _, code := range []string{"400", "401", "403", "404", "409", "422", "503"} {
				if responses[code] == nil {
					responses[code] = map[string]any{"description": "Request validation, authorization, record conflict or dependency failure", "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"oneOf": []any{map[string]any{"$ref": "#/components/schemas/ErrorResponse"}, map[string]any{"type": "object", "required": []string{"error"}, "properties": map[string]any{"error": map[string]any{"type": "string"}, "message": map[string]any{"type": "string"}}, "description": "Legacy authentication error; supported during transition"}}}}}}
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
