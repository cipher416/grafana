package service

import (
	"testing"

	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/stretchr/testify/assert"
)

func TestBuildGrafanaExternalID(t *testing.T) {
	id := buildGrafanaExternalID("stack123", "P7DC3E4760")
	assert.Equal(t, "stack123-P7DC3E4760", id)
	assert.True(t, isValidGrafanaExternalID(id, "stack123", "P7DC3E4760"))
	assert.False(t, isValidGrafanaExternalID(id, "otherstack", "P7DC3E4760"))
	assert.False(t, isValidGrafanaExternalID(id, "stack123", "OTHERUID"))
}

func TestEnsureGrafanaExternalID(t *testing.T) {
	t.Run("generates stack-uid for grafana_assume_role", func(t *testing.T) {
		jd := simplejson.NewFromAny(map[string]any{"authType": grafanaAssumeRoleAuthType})
		ensureGrafanaExternalID("dsUid1", "stackABC", jd, true)
		assert.Equal(t, "stackABC-dsUid1", jd.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("no mint when generation disabled", func(t *testing.T) {
		jd := simplejson.NewFromAny(map[string]any{"authType": grafanaAssumeRoleAuthType})
		ensureGrafanaExternalID("dsUid1", "stackABC", jd, false)
		assert.Empty(t, jd.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("clears stolen ID even when generation disabled", func(t *testing.T) {
		stolen := "stackABC-otherUid"
		jd := simplejson.NewFromAny(map[string]any{
			"authType":               grafanaAssumeRoleAuthType,
			grafanaExternalIDJSONKey: stolen,
		})
		ensureGrafanaExternalID("dsUid1", "stackABC", jd, false)
		assert.Empty(t, jd.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("keeps valid client-supplied ID for pre-save UX", func(t *testing.T) {
		clientID := "stackABC-dsUid1"
		jd := simplejson.NewFromAny(map[string]any{
			"authType":                          grafanaAssumeRoleAuthType,
			usePerDatasourceExternalIDJSONKey:   true,
			grafanaExternalIDJSONKey:            clientID,
		})
		ensureGrafanaExternalID("dsUid1", "stackABC", jd, true)
		assert.Equal(t, clientID, jd.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("replaces ID that belongs to another datasource", func(t *testing.T) {
		stolen := "stackABC-otherUid"
		jd := simplejson.NewFromAny(map[string]any{
			"authType":               grafanaAssumeRoleAuthType,
			grafanaExternalIDJSONKey: stolen,
		})
		ensureGrafanaExternalID("dsUid1", "stackABC", jd, true)
		assert.Equal(t, "stackABC-dsUid1", jd.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("clears stolen ID under non-GAR auth without minting", func(t *testing.T) {
		stolen := "stackABC-otherUid"
		jd := simplejson.NewFromAny(map[string]any{
			"authType":               "keys",
			"externalId":             "cross-account",
			grafanaExternalIDJSONKey: stolen,
		})
		ensureGrafanaExternalID("dsUid1", "stackABC", jd, true)
		assert.Empty(t, jd.Get(grafanaExternalIDJSONKey).MustString())
		assert.Equal(t, "cross-account", jd.Get("externalId").MustString())
	})

	t.Run("no-op for keys auth without grafanaExternalId", func(t *testing.T) {
		jd := simplejson.NewFromAny(map[string]any{"authType": "keys", "externalId": "cross-account"})
		ensureGrafanaExternalID("dsUid1", "stackABC", jd, true)
		assert.Empty(t, jd.Get(grafanaExternalIDJSONKey).MustString())
		assert.Equal(t, "cross-account", jd.Get("externalId").MustString())
	})

	t.Run("no-op when stack external ID missing", func(t *testing.T) {
		jd := simplejson.NewFromAny(map[string]any{"authType": grafanaAssumeRoleAuthType})
		ensureGrafanaExternalID("dsUid1", "", jd, true)
		assert.Empty(t, jd.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("does not mint when usePerDatasourceExternalId is false", func(t *testing.T) {
		jd := simplejson.NewFromAny(map[string]any{
			"authType":                        grafanaAssumeRoleAuthType,
			usePerDatasourceExternalIDJSONKey: false,
			grafanaExternalIDJSONKey:          "",
		})
		ensureGrafanaExternalID("dsUid1", "stackABC", jd, true)
		assert.Empty(t, jd.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("mints when usePerDatasourceExternalId is true", func(t *testing.T) {
		jd := simplejson.NewFromAny(map[string]any{
			"authType":                        grafanaAssumeRoleAuthType,
			usePerDatasourceExternalIDJSONKey: true,
		})
		ensureGrafanaExternalID("dsUid1", "stackABC", jd, true)
		assert.Equal(t, "stackABC-dsUid1", jd.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("mints when bool absent (new DS default per-DS)", func(t *testing.T) {
		jd := simplejson.NewFromAny(map[string]any{
			"authType": grafanaAssumeRoleAuthType,
		})
		ensureGrafanaExternalID("dsUid1", "stackABC", jd, true)
		assert.Equal(t, "stackABC-dsUid1", jd.Get(grafanaExternalIDJSONKey).MustString())
	})
}

func TestPreserveGrafanaExternalID(t *testing.T) {
	t.Run("allows clear when usePerDatasourceExternalId is false", func(t *testing.T) {
		existing := simplejson.NewFromAny(map[string]any{
			"authType":                        grafanaAssumeRoleAuthType,
			usePerDatasourceExternalIDJSONKey: true,
			grafanaExternalIDJSONKey:          "stackABC-dsUid1",
		})
		updated := simplejson.NewFromAny(map[string]any{
			"authType":                        grafanaAssumeRoleAuthType,
			usePerDatasourceExternalIDJSONKey: false,
			grafanaExternalIDJSONKey:          "",
		})
		preserveGrafanaExternalID("dsUid1", "stackABC", existing, updated, true)
		assert.Empty(t, updated.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("preserves when bool omitted even if grafanaExternalId omitted", func(t *testing.T) {
		existing := simplejson.NewFromAny(map[string]any{
			"authType":                        grafanaAssumeRoleAuthType,
			usePerDatasourceExternalIDJSONKey: true,
			grafanaExternalIDJSONKey:          "stackABC-dsUid1",
		})
		updated := simplejson.NewFromAny(map[string]any{
			"authType": grafanaAssumeRoleAuthType,
		})
		preserveGrafanaExternalID("dsUid1", "stackABC", existing, updated, true)
		assert.Equal(t, "stackABC-dsUid1", updated.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("preserves when allowGenerate false and updated clears field", func(t *testing.T) {
		existing := simplejson.NewFromAny(map[string]any{
			"authType":               grafanaAssumeRoleAuthType,
			grafanaExternalIDJSONKey: "stackABC-dsUid1",
		})
		updated := simplejson.NewFromAny(map[string]any{
			"authType":                        grafanaAssumeRoleAuthType,
			usePerDatasourceExternalIDJSONKey: false,
			grafanaExternalIDJSONKey:          "",
		})
		preserveGrafanaExternalID("dsUid1", "stackABC", existing, updated, false)
		assert.Equal(t, "stackABC-dsUid1", updated.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("restores existing after scrubbing stolen update when bool omitted", func(t *testing.T) {
		existing := simplejson.NewFromAny(map[string]any{
			"authType":               grafanaAssumeRoleAuthType,
			grafanaExternalIDJSONKey: "stackABC-dsUid1",
		})
		updated := simplejson.NewFromAny(map[string]any{
			"authType":               grafanaAssumeRoleAuthType,
			grafanaExternalIDJSONKey: "stackABC-otherUid",
		})
		preserveGrafanaExternalID("dsUid1", "stackABC", existing, updated, true)
		assert.Equal(t, "stackABC-dsUid1", updated.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("FT off restores existing after scrubbing stolen update", func(t *testing.T) {
		existing := simplejson.NewFromAny(map[string]any{
			"authType":               grafanaAssumeRoleAuthType,
			grafanaExternalIDJSONKey: "stackABC-dsUid1",
		})
		updated := simplejson.NewFromAny(map[string]any{
			"authType":               grafanaAssumeRoleAuthType,
			grafanaExternalIDJSONKey: "stackABC-otherUid",
		})
		preserveGrafanaExternalID("dsUid1", "stackABC", existing, updated, false)
		assert.Equal(t, "stackABC-dsUid1", updated.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("preserves existing ID when generation disabled", func(t *testing.T) {
		existing := simplejson.NewFromAny(map[string]any{
			"authType":               grafanaAssumeRoleAuthType,
			grafanaExternalIDJSONKey: "stackABC-dsUid1",
		})
		updated := simplejson.NewFromAny(map[string]any{
			"authType":               grafanaAssumeRoleAuthType,
			grafanaExternalIDJSONKey: "stackABC-stolen",
		})
		preserveGrafanaExternalID("dsUid1", "stackABC", existing, updated, false)
		assert.Equal(t, "stackABC-dsUid1", updated.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("clears planted ID on legacy GAR update without minting", func(t *testing.T) {
		existing := simplejson.NewFromAny(map[string]any{"authType": grafanaAssumeRoleAuthType})
		updated := simplejson.NewFromAny(map[string]any{
			"authType":               grafanaAssumeRoleAuthType,
			grafanaExternalIDJSONKey: "stackABC-otherUid",
		})
		preserveGrafanaExternalID("dsUid1", "stackABC", existing, updated, true)
		assert.Empty(t, updated.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("mints when legacy GAR explicitly opts in", func(t *testing.T) {
		existing := simplejson.NewFromAny(map[string]any{"authType": grafanaAssumeRoleAuthType})
		updated := simplejson.NewFromAny(map[string]any{
			"authType":                        grafanaAssumeRoleAuthType,
			usePerDatasourceExternalIDJSONKey: true,
		})
		preserveGrafanaExternalID("dsUid1", "stackABC", existing, updated, true)
		assert.Equal(t, "stackABC-dsUid1", updated.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("scrubs invalid ID already stored and does not re-preserve it", func(t *testing.T) {
		existing := simplejson.NewFromAny(map[string]any{
			"authType":               grafanaAssumeRoleAuthType,
			grafanaExternalIDJSONKey: "stackABC-otherUid",
		})
		updated := simplejson.NewFromAny(map[string]any{
			"authType": grafanaAssumeRoleAuthType,
		})
		preserveGrafanaExternalID("dsUid1", "stackABC", existing, updated, true)
		assert.Empty(t, updated.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("clears smuggled ID under keys then mints on switch to GAR", func(t *testing.T) {
		existing := simplejson.NewFromAny(map[string]any{
			"authType":               "keys",
			grafanaExternalIDJSONKey: "stackABC-otherUid",
		})
		updated := simplejson.NewFromAny(map[string]any{
			"authType":               grafanaAssumeRoleAuthType,
			grafanaExternalIDJSONKey: "stackABC-otherUid",
		})
		preserveGrafanaExternalID("dsUid1", "stackABC", existing, updated, true)
		assert.Equal(t, "stackABC-dsUid1", updated.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("clears smuggled ID on switch when generation disabled", func(t *testing.T) {
		existing := simplejson.NewFromAny(map[string]any{
			"authType":               "keys",
			grafanaExternalIDJSONKey: "stackABC-otherUid",
		})
		updated := simplejson.NewFromAny(map[string]any{
			"authType":               grafanaAssumeRoleAuthType,
			grafanaExternalIDJSONKey: "stackABC-otherUid",
		})
		preserveGrafanaExternalID("dsUid1", "stackABC", existing, updated, false)
		assert.Empty(t, updated.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("generates when switching to grafana_assume_role without touching externalId", func(t *testing.T) {
		existing := simplejson.NewFromAny(map[string]any{
			"authType":   "keys",
			"externalId": "cross-account-id",
		})
		updated := simplejson.NewFromAny(map[string]any{
			"authType":   grafanaAssumeRoleAuthType,
			"externalId": "cross-account-id",
		})
		preserveGrafanaExternalID("dsUid1", "stackABC", existing, updated, true)
		assert.Equal(t, "stackABC-dsUid1", updated.Get(grafanaExternalIDJSONKey).MustString())
		assert.Equal(t, "cross-account-id", updated.Get("externalId").MustString())
	})

	t.Run("does not mint on auth switch when usePerDatasourceExternalId is false", func(t *testing.T) {
		existing := simplejson.NewFromAny(map[string]any{
			"authType":   "keys",
			"externalId": "cross-account-id",
		})
		updated := simplejson.NewFromAny(map[string]any{
			"authType":                        grafanaAssumeRoleAuthType,
			usePerDatasourceExternalIDJSONKey: false,
			"externalId":                      "cross-account-id",
		})
		preserveGrafanaExternalID("dsUid1", "stackABC", existing, updated, true)
		assert.Empty(t, updated.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("does not generate on auth switch when generation disabled", func(t *testing.T) {
		existing := simplejson.NewFromAny(map[string]any{
			"authType":   "keys",
			"externalId": "cross-account-id",
		})
		updated := simplejson.NewFromAny(map[string]any{
			"authType":   grafanaAssumeRoleAuthType,
			"externalId": "cross-account-id",
		})
		preserveGrafanaExternalID("dsUid1", "stackABC", existing, updated, false)
		assert.Empty(t, updated.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("clears field when switching away from grafana_assume_role with FT on", func(t *testing.T) {
		existing := simplejson.NewFromAny(map[string]any{
			"authType":               grafanaAssumeRoleAuthType,
			grafanaExternalIDJSONKey: "stackABC-dsUid1",
		})
		updated := simplejson.NewFromAny(map[string]any{
			"authType": "keys",
		})
		preserveGrafanaExternalID("dsUid1", "stackABC", existing, updated, true)
		assert.Empty(t, updated.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("preserves field when switching away from grafana_assume_role with FT off", func(t *testing.T) {
		existing := simplejson.NewFromAny(map[string]any{
			"authType":               grafanaAssumeRoleAuthType,
			grafanaExternalIDJSONKey: "stackABC-dsUid1",
		})
		updated := simplejson.NewFromAny(map[string]any{
			"authType": "keys",
		})
		preserveGrafanaExternalID("dsUid1", "stackABC", existing, updated, false)
		assert.Equal(t, "stackABC-dsUid1", updated.Get(grafanaExternalIDJSONKey).MustString())
	})

	t.Run("does not auto-migrate legacy grafana_assume_role on update", func(t *testing.T) {
		existing := simplejson.NewFromAny(map[string]any{"authType": grafanaAssumeRoleAuthType})
		updated := simplejson.NewFromAny(map[string]any{"authType": grafanaAssumeRoleAuthType, "assumeRoleArn": "arn:aws:iam::123:role/x"})
		preserveGrafanaExternalID("dsUid1", "stackABC", existing, updated, true)
		assert.Empty(t, updated.Get(grafanaExternalIDJSONKey).MustString())
	})
}
