package service

import (
	"github.com/grafana/grafana/pkg/components/simplejson"
)

const (
	grafanaAssumeRoleAuthType          = "grafana_assume_role"
	grafanaExternalIDJSONKey           = "grafanaExternalId"
	usePerDatasourceExternalIDJSONKey  = "usePerDatasourceExternalId"
)

// buildGrafanaExternalID returns "{stackExternalId}-{dsUID}".
// The stack prefix keeps IDs unique across Cloud stacks (UIDs alone are only unique per org).
// The datasource UID binds the ID to this datasource so a copied ID from another DS fails validation.
func buildGrafanaExternalID(stackExternalID, datasourceUID string) string {
	return stackExternalID + "-" + datasourceUID
}

// isValidGrafanaExternalID reports whether id is bound to this stack + datasource UID.
// Equality is used instead of splitting on "-" because stack IDs and UIDs may contain dashes.
func isValidGrafanaExternalID(id, stackExternalID, datasourceUID string) bool {
	if id == "" || stackExternalID == "" || datasourceUID == "" {
		return false
	}
	return id == buildGrafanaExternalID(stackExternalID, datasourceUID)
}

// usePerDatasourceExternalID reports whether jsonData sets usePerDatasourceExternalId and its value.
func usePerDatasourceExternalID(jsonData *simplejson.Json) (set bool, enabled bool) {
	if jsonData == nil {
		return false, false
	}
	v, exists := jsonData.CheckGet(usePerDatasourceExternalIDJSONKey)
	if !exists {
		return false, false
	}
	return true, v.MustBool()
}

// clearInvalidGrafanaExternalID removes a client- or store-supplied grafanaExternalId that is
// not bound to this datasource. Dual-read always prefers a set grafanaExternalId for STS, so
// invalid values must never be persisted regardless of the minting feature toggle.
func clearInvalidGrafanaExternalID(uid, stackExternalID string, jsonData *simplejson.Json) {
	if jsonData == nil {
		return
	}
	id := jsonData.Get(grafanaExternalIDJSONKey).MustString()
	if id == "" {
		return
	}
	if !isValidGrafanaExternalID(id, stackExternalID, uid) {
		jsonData.Del(grafanaExternalIDJSONKey)
	}
}

// ensureGrafanaExternalID validates (and clears) any client-supplied grafanaExternalId on create.
// When allowGenerate is true and auth is grafana_assume_role:
//   - usePerDatasourceExternalId=false → stack mode (clear ID, no mint)
//   - true or unset → mint when empty (new datasources default to per-DS)
// Valid client-supplied IDs for this stack+uid are kept.
func ensureGrafanaExternalID(uid, stackExternalID string, jsonData *simplejson.Json, allowGenerate bool) {
	if jsonData == nil {
		return
	}

	clearInvalidGrafanaExternalID(uid, stackExternalID, jsonData)

	if !allowGenerate {
		return
	}
	if jsonData.Get("authType").MustString() != grafanaAssumeRoleAuthType {
		return
	}

	modeSet, modeOn := usePerDatasourceExternalID(jsonData)
	if modeSet && !modeOn {
		jsonData.Del(grafanaExternalIDJSONKey)
		return
	}

	if stackExternalID == "" || uid == "" {
		return
	}
	if jsonData.Get(grafanaExternalIDJSONKey).MustString() != "" {
		return
	}

	jsonData.Set(grafanaExternalIDJSONKey, buildGrafanaExternalID(stackExternalID, uid))
}

// preserveGrafanaExternalID keeps a valid existing per-datasource external ID across updates unless
// the caller explicitly sets usePerDatasourceExternalId=false (stack mode) while allowGenerate is
// true. Invalid IDs are always scrubbed. When allowGenerate is true it may mint when switching into
// grafana_assume_role unless stack mode is requested.
//
// Omitting usePerDatasourceExternalId on update preserves an existing ID (Terraform-friendly).
// Legacy GAR datasources without an ID keep the stack fallback until they opt in.
func preserveGrafanaExternalID(uid, stackExternalID string, existing, updated *simplejson.Json, allowGenerate bool) {
	if updated == nil {
		return
	}

	// Never persist a stolen / mismatched ID from the update payload.
	clearInvalidGrafanaExternalID(uid, stackExternalID, updated)

	existingID := ""
	existingAuthType := ""
	if existing != nil {
		existingID = existing.Get(grafanaExternalIDJSONKey).MustString()
		existingAuthType = existing.Get("authType").MustString()
	}

	updatedAuthType := updated.Get("authType").MustString()
	modeSet, modeOn := usePerDatasourceExternalID(updated)

	// Leaving Grafana Assume Role: drop the GAR-specific ID when minting/clear is FT-enabled.
	if allowGenerate && updatedAuthType != grafanaAssumeRoleAuthType {
		updated.Del(grafanaExternalIDJSONKey)
		return
	}

	if allowGenerate && modeSet && !modeOn {
		updated.Del(grafanaExternalIDJSONKey)
		return
	}

	if existingID != "" {
		if isValidGrafanaExternalID(existingID, stackExternalID, uid) {
			// Immutable value while remaining in per-DS mode; restore if omitted/cleared without explicit false.
			updated.Set(grafanaExternalIDJSONKey, existingID)
			return
		}
		updated.Del(grafanaExternalIDJSONKey)
	}

	if !allowGenerate {
		return
	}
	if updatedAuthType != grafanaAssumeRoleAuthType {
		return
	}
	if stackExternalID == "" || uid == "" {
		return
	}
	if updated.Get(grafanaExternalIDJSONKey).MustString() != "" {
		return
	}

	// Mint when switching into GAR (bool unset defaults to per-DS) or when explicitly opting in.
	switchingIn := existingAuthType != grafanaAssumeRoleAuthType
	optingIn := modeSet && modeOn
	if !switchingIn && !optingIn {
		return
	}

	updated.Set(grafanaExternalIDJSONKey, buildGrafanaExternalID(stackExternalID, uid))
}
