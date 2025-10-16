package services

import (
	"context"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/valpere/shopogoda/tests/helpers"
)

func TestNewLocalizationService(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	service := NewLocalizationService(logger)

	assert.NotNil(t, service)
	assert.NotNil(t, service.translations)
	assert.Equal(t, "en-US", service.defaultLanguage)
	assert.NotNil(t, service.logger)
}

func TestLocalizationService_LoadTranslations(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	service := NewLocalizationService(logger)

	t.Run("successful load", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"languages.json": &fstest.MapFile{
				Data: []byte(`{
					"en-US": {"code": "en-US", "name": "English", "flag": "🇺🇸"},
					"uk-UA": {"code": "uk-UA", "name": "Ukrainian", "flag": "🇺🇦"}
				}`),
			},
			"en-US.json": &fstest.MapFile{
				Data: []byte(`{
					"greeting": "Hello",
					"weather": "Weather",
					"temperature": "Temperature: %.1f°C"
				}`),
			},
			"uk-UA.json": &fstest.MapFile{
				Data: []byte(`{
					"greeting": "Привіт",
					"weather": "Погода",
					"temperature": "Температура: %.1f°C"
				}`),
			},
		}

		err := service.LoadTranslations(mockFS)
		require.NoError(t, err)

		assert.Len(t, service.supportedLanguages, 2)
		assert.Len(t, service.translations, 2)

		enLang, exists := service.supportedLanguages["en-US"]
		assert.True(t, exists)
		assert.Equal(t, "English", enLang.Name)
		assert.Equal(t, "🇺🇸", enLang.Flag)

		ukLang, exists := service.supportedLanguages["uk-UA"]
		assert.True(t, exists)
		assert.Equal(t, "Ukrainian", ukLang.Name)
		assert.Equal(t, "🇺🇦", ukLang.Flag)
	})

	t.Run("missing languages.json", func(t *testing.T) {
		mockFS := fstest.MapFS{}
		err := service.LoadTranslations(mockFS)
		assert.Error(t, err)
	})

	t.Run("invalid languages.json format", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"languages.json": &fstest.MapFile{
				Data: []byte(`invalid json`),
			},
		}
		err := service.LoadTranslations(mockFS)
		assert.Error(t, err)
	})

	t.Run("missing translation file", func(t *testing.T) {
		service := NewLocalizationService(logger)
		mockFS := fstest.MapFS{
			"languages.json": &fstest.MapFile{
				Data: []byte(`{
					"en-US": {"code": "en-US", "name": "English", "flag": "🇺🇸"}
				}`),
			},
			// Missing en-US.json file
		}

		err := service.LoadTranslations(mockFS)
		require.NoError(t, err) // Should not error, just log

		// English should be in supported languages but not have translations
		assert.Len(t, service.supportedLanguages, 1)
		assert.Len(t, service.translations, 0)
	})

	t.Run("invalid translation file format", func(t *testing.T) {
		service := NewLocalizationService(logger)
		mockFS := fstest.MapFS{
			"languages.json": &fstest.MapFile{
				Data: []byte(`{
					"en-US": {"code": "en-US", "name": "English", "flag": "🇺🇸"}
				}`),
			},
			"en-US.json": &fstest.MapFile{
				Data: []byte(`invalid json`),
			},
		}

		err := service.LoadTranslations(mockFS)
		require.NoError(t, err) // Should not error, just log

		// English should be in supported languages but not have translations
		assert.Len(t, service.supportedLanguages, 1)
		assert.Len(t, service.translations, 0)
	})
}

func TestLocalizationService_T(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	service := NewLocalizationService(logger)

	// Setup test translations
	mockFS := fstest.MapFS{
		"languages.json": &fstest.MapFile{
			Data: []byte(`{
				"en-US": {"code": "en-US", "name": "English", "flag": "🇺🇸"},
				"uk-UA": {"code": "uk-UA", "name": "Ukrainian", "flag": "🇺🇦"}
			}`),
		},
		"en-US.json": &fstest.MapFile{
			Data: []byte(`{
				"greeting": "Hello",
				"weather": "Weather",
				"temperature": "Temperature: %.1f°C",
				"forecast_days": "Forecast for %d days"
			}`),
		},
		"uk-UA.json": &fstest.MapFile{
			Data: []byte(`{
				"greeting": "Привіт",
				"weather": "Погода"
			}`),
		},
	}

	err := service.LoadTranslations(mockFS)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("translate existing key", func(t *testing.T) {
		result := service.T(ctx, "en-US", "greeting")
		assert.Equal(t, "Hello", result)
	})

	t.Run("translate with arguments", func(t *testing.T) {
		result := service.T(ctx, "en-US", "temperature", 25.5)
		assert.Equal(t, "Temperature: 25.5°C", result)
	})

	t.Run("translate with multiple arguments", func(t *testing.T) {
		result := service.T(ctx, "en-US", "forecast_days", 5)
		assert.Equal(t, "Forecast for 5 days", result)
	})

	t.Run("fallback to default language", func(t *testing.T) {
		// temperature key doesn't exist in Ukrainian, should fall back to English
		result := service.T(ctx, "uk-UA", "temperature", 20.0)
		assert.Equal(t, "Temperature: 20.0°C", result)
	})

	t.Run("return key when translation not found", func(t *testing.T) {
		result := service.T(ctx, "en-US", "nonexistent_key")
		assert.Equal(t, "nonexistent_key", result)
	})

	t.Run("unsupported language falls back to default", func(t *testing.T) {
		// French not supported, should fall back to English default
		result := service.T(ctx, "fr-FR", "greeting")
		assert.Equal(t, "Hello", result)
	})

	t.Run("existing key in requested language", func(t *testing.T) {
		result := service.T(ctx, "uk-UA", "greeting")
		assert.Equal(t, "Привіт", result)
	})
}

func TestLocalizationService_IsLanguageSupported(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	service := NewLocalizationService(logger)

	mockFS := fstest.MapFS{
		"languages.json": &fstest.MapFile{
			Data: []byte(`{
				"en-US": {"code": "en-US", "name": "English", "flag": "🇺🇸"},
				"uk-UA": {"code": "uk-UA", "name": "Ukrainian", "flag": "🇺🇦"}
			}`),
		},
		"en-US.json": &fstest.MapFile{
			Data: []byte(`{"greeting": "Hello"}`),
		},
		"uk-UA.json": &fstest.MapFile{
			Data: []byte(`{"greeting": "Привіт"}`),
		},
	}

	err := service.LoadTranslations(mockFS)
	require.NoError(t, err)

	t.Run("supported language", func(t *testing.T) {
		assert.True(t, service.IsLanguageSupported("en-US"))
		assert.True(t, service.IsLanguageSupported("uk-UA"))
	})

	t.Run("unsupported language", func(t *testing.T) {
		assert.False(t, service.IsLanguageSupported("fr-FR"))
		assert.False(t, service.IsLanguageSupported("de-DE"))
		assert.False(t, service.IsLanguageSupported(""))
	})
}

func TestLocalizationService_GetSupportedLanguages(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	service := NewLocalizationService(logger)

	mockFS := fstest.MapFS{
		"languages.json": &fstest.MapFile{
			Data: []byte(`{
				"en-US": {"code": "en-US", "name": "English", "flag": "🇺🇸"},
				"uk-UA": {"code": "uk-UA", "name": "Ukrainian", "flag": "🇺🇦"},
				"de-DE": {"code": "de-DE", "name": "German", "flag": "🇩🇪"}
			}`),
		},
		"en-US.json": &fstest.MapFile{Data: []byte(`{"key": "value"}`)},
		"uk-UA.json": &fstest.MapFile{Data: []byte(`{"key": "value"}`)},
		"de-DE.json": &fstest.MapFile{Data: []byte(`{"key": "value"}`)},
	}

	err := service.LoadTranslations(mockFS)
	require.NoError(t, err)

	languages := service.GetSupportedLanguages()
	assert.Len(t, languages, 3)
	assert.Contains(t, languages, "en-US")
	assert.Contains(t, languages, "uk-UA")
	assert.Contains(t, languages, "de-DE")
}

func TestLocalizationService_GetLanguageByCode(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	service := NewLocalizationService(logger)

	mockFS := fstest.MapFS{
		"languages.json": &fstest.MapFile{
			Data: []byte(`{
				"en-US": {"code": "en-US", "name": "English", "flag": "🇺🇸"},
				"uk-UA": {"code": "uk-UA", "name": "Ukrainian", "flag": "🇺🇦"}
			}`),
		},
		"en-US.json": &fstest.MapFile{Data: []byte(`{"key": "value"}`)},
		"uk-UA.json": &fstest.MapFile{Data: []byte(`{"key": "value"}`)},
	}

	err := service.LoadTranslations(mockFS)
	require.NoError(t, err)

	t.Run("existing language code", func(t *testing.T) {
		lang, exists := service.GetLanguageByCode("en-US")
		assert.True(t, exists)
		assert.Equal(t, "en-US", lang.Code)
		assert.Equal(t, "English", lang.Name)
		assert.Equal(t, "🇺🇸", lang.Flag)
	})

	t.Run("another existing language code", func(t *testing.T) {
		lang, exists := service.GetLanguageByCode("uk-UA")
		assert.True(t, exists)
		assert.Equal(t, "uk-UA", lang.Code)
		assert.Equal(t, "Ukrainian", lang.Name)
	})

	t.Run("nonexistent language code returns default", func(t *testing.T) {
		lang, exists := service.GetLanguageByCode("fr-FR")
		assert.False(t, exists)
		assert.Equal(t, "en-US", lang.Code) // Should return default language
	})
}

func TestLocalizationService_DetectLanguageFromName(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	service := NewLocalizationService(logger)

	mockFS := fstest.MapFS{
		"languages.json": &fstest.MapFile{
			Data: []byte(`{
				"en-US": {"code": "en-US", "name": "English", "flag": "🇺🇸"},
				"uk-UA": {"code": "uk-UA", "name": "Ukrainian", "flag": "🇺🇦"}
			}`),
		},
		"en-US.json": &fstest.MapFile{Data: []byte(`{"key": "value"}`)},
		"uk-UA.json": &fstest.MapFile{Data: []byte(`{"key": "value"}`)},
	}

	err := service.LoadTranslations(mockFS)
	require.NoError(t, err)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"english lowercase", "english", "en-US"},
		{"english capitalized", "English", "en-US"},
		{"ukrainian", "ukrainian", "uk-UA"},
		{"ukrainian cyrillic", "українська", "uk-UA"},
		{"german", "german", "de-DE"},
		{"deutsch", "deutsch", "de-DE"},
		{"french", "french", "fr-FR"},
		{"français", "français", "fr-FR"},
		{"spanish", "spanish", "es-ES"},
		{"español", "español", "es-ES"},
		{"full language code en-US", "en-US", "en-US"},
		{"full language code uk-UA", "uk-UA", "uk-UA"},
		{"unknown language", "unknown", "en-US"},
		{"empty string", "", "en-US"},
		{"whitespace", "  ", "en-US"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.DetectLanguageFromName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLocalizationService_GetAvailableTranslationKeys(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	service := NewLocalizationService(logger)

	mockFS := fstest.MapFS{
		"languages.json": &fstest.MapFile{
			Data: []byte(`{
				"en-US": {"code": "en-US", "name": "English", "flag": "🇺🇸"},
				"uk-UA": {"code": "uk-UA", "name": "Ukrainian", "flag": "🇺🇦"}
			}`),
		},
		"en-US.json": &fstest.MapFile{
			Data: []byte(`{
				"greeting": "Hello",
				"weather": "Weather",
				"temperature": "Temperature"
			}`),
		},
		"uk-UA.json": &fstest.MapFile{
			Data: []byte(`{
				"greeting": "Привіт",
				"weather": "Погода"
			}`),
		},
	}

	err := service.LoadTranslations(mockFS)
	require.NoError(t, err)

	t.Run("get keys for existing language", func(t *testing.T) {
		keys := service.GetAvailableTranslationKeys("en-US")
		assert.Len(t, keys, 3)
		assert.Contains(t, keys, "greeting")
		assert.Contains(t, keys, "weather")
		assert.Contains(t, keys, "temperature")
	})

	t.Run("get keys for another language", func(t *testing.T) {
		keys := service.GetAvailableTranslationKeys("uk-UA")
		assert.Len(t, keys, 2)
		assert.Contains(t, keys, "greeting")
		assert.Contains(t, keys, "weather")
	})

	t.Run("get keys for nonexistent language", func(t *testing.T) {
		keys := service.GetAvailableTranslationKeys("fr-FR")
		assert.Nil(t, keys)
	})
}

func TestLocalizationService_ConcurrentAccess(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	service := NewLocalizationService(logger)

	mockFS := fstest.MapFS{
		"languages.json": &fstest.MapFile{
			Data: []byte(`{
				"en-US": {"code": "en-US", "name": "English", "flag": "🇺🇸"}
			}`),
		},
		"en-US.json": &fstest.MapFile{
			Data: []byte(`{
				"greeting": "Hello",
				"weather": "Weather"
			}`),
		},
	}

	err := service.LoadTranslations(mockFS)
	require.NoError(t, err)

	t.Run("concurrent reads", func(t *testing.T) {
		var wg sync.WaitGroup
		ctx := context.Background()

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = service.T(ctx, "en-US", "greeting")
				_ = service.IsLanguageSupported("en-US")
				_ = service.GetAvailableTranslationKeys("en-US")
			}()
		}

		wg.Wait()
	})

	t.Run("concurrent read while loading", func(t *testing.T) {
		var wg sync.WaitGroup
		ctx := context.Background()

		// Start reading
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = service.T(ctx, "en-US", "greeting")
			}()
		}

		// Reload translations concurrently
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = service.LoadTranslations(mockFS)
		}()

		wg.Wait()
	})
}

func TestLocalizationService_EmptyTranslations(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	service := NewLocalizationService(logger)

	mockFS := fstest.MapFS{
		"languages.json": &fstest.MapFile{
			Data: []byte(`{
				"en-US": {"code": "en-US", "name": "English", "flag": "🇺🇸"}
			}`),
		},
		"en-US.json": &fstest.MapFile{
			Data: []byte(`{}`), // Empty translations
		},
	}

	err := service.LoadTranslations(mockFS)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("translate with empty translations", func(t *testing.T) {
		result := service.T(ctx, "en-US", "any_key")
		assert.Equal(t, "any_key", result) // Should return key
	})

	t.Run("get keys returns empty slice", func(t *testing.T) {
		keys := service.GetAvailableTranslationKeys("en-US")
		assert.Empty(t, keys)
	})
}
