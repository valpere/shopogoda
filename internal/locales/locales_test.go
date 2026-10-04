package locales

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Activity counters live in memory and reset on restart, so their labels must
// not claim a 24h window.
var sinceStartKeys = []string{
	"admin_detailed_stats_messages_sent",
	"admin_detailed_stats_weather_requests",
	"admin_recent_activity_messages",
	"admin_recent_activity_weather_requests",
	"admin_stats_messages_sent",
	"admin_stats_weather_requests",
	"admin_users_messages",
	"admin_users_weather_requests",
}

func TestActivityCounterLabelsAreNot24h(t *testing.T) {
	files, err := LocalesFS.ReadDir(".")
	require.NoError(t, err)

	checked := 0
	for _, f := range files {
		if filepath.Ext(f.Name()) != ".json" || f.Name() == "languages.json" {
			continue
		}
		data, err := LocalesFS.ReadFile(f.Name())
		require.NoError(t, err)
		var msgs map[string]string
		require.NoError(t, json.Unmarshal(data, &msgs), f.Name())

		for _, key := range sinceStartKeys {
			val, ok := msgs[key]
			require.True(t, ok, "%s missing %s", f.Name(), key)
			assert.False(t, strings.Contains(val, "24h") || strings.Contains(val, "24г"), "%s %s still says 24h: %q", f.Name(), key, val)
			assert.Contains(t, val, "(", "%s %s should carry a since-start qualifier: %q", f.Name(), key, val)
		}
		checked++
	}
	assert.Equal(t, 5, checked, "en-US, uk-UA, de-DE, es-ES, fr-FR")
}
