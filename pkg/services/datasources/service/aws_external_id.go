package service

import (
	"github.com/grafana/grafana/pkg/components/simplejson"
)

const (
	grafanaAssumeRoleAuthType = "grafana_assume_role"
	grafanaExternalIDJSONKey  = "grafanaExternalId"
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
// When allowGenerate is true and auth is grafana_assume_role, it mints {stack}-{uid} only when the
// JSON key is absent. A present key (even empty) means explicit stack mode and is not reminted.
// If the client already supplied a valid ID (pre-save UX), it is kept.
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
	if stackExternalID == "" || uid == "" {
		return
	}

	// Key present (even if "") = caller chose stack mode; do not remint.
	if _, exists := jsonData.CheckGet(grafanaExternalIDJSONKey); exists {
		return
	}

	jsonData.Set(grafanaExternalIDJSONKey, buildGrafanaExternalID(stackExternalID, uid))
}

// preserveGrafanaExternalID keeps a valid existing per-datasource external ID immutable across updates
// (unless allowGenerate allows clearing back to stack mode), scrubs invalid stored or client-supplied
// values, and optionally mints when switching into grafana_assume_role (when allowGenerate is true).
//
// Legacy grafana_assume_role datasources without an ID keep using the stack-level fallback until
// explicitly migrated — we do not generate on ordinary updates.
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

	if existingID != "" {
		if isValidGrafanaExternalID(existingID, stackExternalID, uid) {
			updatedID := updated.Get(grafanaExternalIDJSONKey).MustString()
			if updatedID == "" {
				if allowGenerate {
					// UI toggled to stack mode (or stolen id scrubbed to empty with FT on).
					return
				}
				updated.Set(grafanaExternalIDJSONKey, existingID)
				return
			}
			// Non-empty after scrub must be the bound value; force existing (immutable value while per-DS).
			updated.Set(grafanaExternalIDJSONKey, existingID)
			return
		}
		updated.Del(grafanaExternalIDJSONKey)
	}

	if !allowGenerate {
		return
	}

	updatedAuthType := updated.Get("authType").MustString()
	if updatedAuthType != grafanaAssumeRoleAuthType {
		return
	}

	// Generate only when switching into grafana_assume_role from another auth type.
	if existingAuthType == grafanaAssumeRoleAuthType {
		return
	}
	if stackExternalID == "" || uid == "" {
		return
	}

	if updated.Get(grafanaExternalIDJSONKey).MustString() != "" {
		return
	}

	updated.Set(grafanaExternalIDJSONKey, buildGrafanaExternalID(stackExternalID, uid))
}
