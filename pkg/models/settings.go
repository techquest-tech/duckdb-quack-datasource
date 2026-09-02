package models

import (
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// PluginSettings is the datasource configuration persisted in jsonData
// (endpoint / tablePrefix / queryTimeoutMS) plus secrets (token).
type PluginSettings struct {
	Endpoint       string                `json:"endpoint"`
	TablePrefix    string                `json:"tablePrefix"`
	QueryTimeoutMS int64                 `json:"queryTimeoutMS"`
	Secrets        *SecretPluginSettings `json:"-"`
}

// SecretPluginSettings holds the Quack auth token (stored in secureJsonData).
type SecretPluginSettings struct {
	Token string `json:"token"`
}

// LoadPluginSettings parses jsonData + secureJsonData with defaults applied.
func LoadPluginSettings(source backend.DataSourceInstanceSettings) (*PluginSettings, error) {
	settings := PluginSettings{
		Endpoint:       "localhost:9494",
		QueryTimeoutMS: 30000,
	}
	if len(source.JSONData) > 0 {
		if err := json.Unmarshal(source.JSONData, &settings); err != nil {
			return nil, fmt.Errorf("could not unmarshal PluginSettings json: %w", err)
		}
	}
	if settings.Endpoint == "" {
		settings.Endpoint = "localhost:9494"
	}
	if settings.QueryTimeoutMS <= 0 {
		settings.QueryTimeoutMS = 30000
	}
	settings.Secrets = loadSecretPluginSettings(source.DecryptedSecureJSONData)
	return &settings, nil
}

func loadSecretPluginSettings(source map[string]string) *SecretPluginSettings {
	return &SecretPluginSettings{
		Token: source["token"],
	}
}
