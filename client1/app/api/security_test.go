//go:build js && wasm

package api

import (
	"encoding/json"
	"testing"
)

func TestSecurityProjectionsExcludeReadOnlyFields(t *testing.T) {
	for _, payload := range []any{SecurityGrantPayload(GroupResourceDTO{ID: 1, GroupID: 2, ResourceID: 3, AccessLevelID: 4, AccessScopeID: 5, Group: "private-display"}), MembershipPayload(UserGroupDTO{ID: 1, UserID: 2, GroupID: 3, User: "private-display"})} {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]any
		json.Unmarshal(data, &fields)
		for _, key := range []string{"id", "created", "modified", "user_name", "group_name", "actor"} {
			if _, ok := fields[key]; ok {
				t.Fatal("server/display field transmitted", key)
			}
		}
	}
}
