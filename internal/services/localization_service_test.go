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
	assert.Equal(t, "en", service.defaultLanguage)
	assert.NotNil(t, service.logger)
}

func TestLocalizationService_LoadTranslations(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	service := NewLocalizationService(logger)

	t.Run("successful load", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"languages.json": &fstest.MapFile{
				Data: []byte(`{
					"en": {"code": "en", "name": "English", "flag": "🇬🇧"},
					"uk": {"code": "uk", "name": "Ukrainian", "flag": "🇺🇦"}
				}`),
			},
			"en.json": &fstest.MapFile{
				Data: []byte(`{
					"greeting": "Hello",
					"weather": "Weather",
					"temperature": "Temperature: %.1f°C"
				}`),
			},
			"uk.json": &fstest.MapFile{
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

		enLang, exists := service.supportedLanguages["en"]
		assert.True(t, exists)
		assert.Equal(t, "English", enLang.Name)
		assert.Equal(t, "🇬🇧", enLang.Flag)

		ukLang, exists := service.supportedLanguages["uk"]
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
					"en": {"code": "en", "name": "English", "flag": "🇬🇧"}
				}`),
			},
			// Missing en.json file
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
					"en": {"code": "en", "name": "English", "flag": "🇬🇧"}
				}`),
			},
			"en.json": &fstest.MapFile{
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
				"en": {"code": "en", "name": "English", "flag": "🇬🇧"},
				"uk": {"code": "uk", "name": "Ukrainian", "flag": "🇺🇦"}
			}`),
		},
		"en.json": &fstest.MapFile{
			Data: []byte(`{
				"greeting": "Hello",
				"weather": "Weather",
				"temperature": "Temperature: %.1f°C",
				"forecast_days": "Forecast for %d days"
			}`),
		},
		"uk.json": &fstest.MapFile{
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
		result := service.T(ctx, "en", "greeting")
		assert.Equal(t, "Hello", result)
	})

	t.Run("translate with arguments", func(t *testing.T) {
		result := service.T(ctx, "en", "temperature", 25.5)
		assert.Equal(t, "Temperature: 25.5°C", result)
	})

	t.Run("translate with multiple arguments", func(t *testing.T) {
		result := service.T(ctx, "en", "forecast_days", 5)
		assert.Equal(t, "Forecast for 5 days", result)
	})

	t.Run("fallback to default language", func(t *testing.T) {
		// temperature key doesn't exist in Ukrainian, should fall back to English
		result := service.T(ctx, "uk", "temperature", 20.0)
		assert.Equal(t, "Temperature: 20.0°C", result)
	})

	t.Run("return key when translation not found", func(t *testing.T) {
		result := service.T(ctx, "en", "nonexistent_key")
		assert.Equal(t, "nonexistent_key", result)
	})

	t.Run("unsupported language falls back to default", func(t *testing.T) {
		// French not supported, should fall back to English default
		result := service.T(ctx, "fr", "greeting")
		assert.Equal(t, "Hello", result)
	})

	t.Run("existing key in requested language", func(t *testing.T) {
		result := service.T(ctx, "uk", "greeting")
		assert.Equal(t, "Привіт", result)
	})
}

func TestLocalizationService_IsLanguageSupported(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	service := NewLocalizationService(logger)

	mockFS := fstest.MapFS{
		"languages.json": &fstest.MapFile{
			Data: []byte(`{
				"en": {"code": "en", "name": "English", "flag": "🇬🇧"},
				"uk": {"code": "uk", "name": "Ukrainian", "flag": "🇺🇦"}
			}`),
		},
		"en.json": &fstest.MapFile{
			Data: []byte(`{"greeting": "Hello"}`),
		},
		"uk.json": &fstest.MapFile{
			Data: []byte(`{"greeting": "Привіт"}`),
		},
	}

	err := service.LoadTranslations(mockFS)
	require.NoError(t, err)

	t.Run("supported language", func(t *testing.T) {
		assert.True(t, service.IsLanguageSupported("en"))
		assert.True(t, service.IsLanguageSupported("uk"))
	})

	t.Run("unsupported language", func(t *testing.T) {
		assert.False(t, service.IsLanguageSupported("fr"))
		assert.False(t, service.IsLanguageSupported("de"))
		assert.False(t, service.IsLanguageSupported(""))
	})
}

func TestLocalizationService_GetSupportedLanguages(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	service := NewLocalizationService(logger)

	mockFS := fstest.MapFS{
		"languages.json": &fstest.MapFile{
			Data: []byte(`{
				"en": {"code": "en", "name": "English", "flag": "🇬🇧"},
				"uk": {"code": "uk", "name": "Ukrainian", "flag": "🇺🇦"},
				"de": {"code": "de", "name": "German", "flag": "🇩🇪"}
			}`),
		},
		"en.json": &fstest.MapFile{Data: []byte(`{"key": "value"}`)},
		"uk.json": &fstest.MapFile{Data: []byte(`{"key": "value"}`)},
		"de.json": &fstest.MapFile{Data: []byte(`{"key": "value"}`)},
	}

	err := service.LoadTranslations(mockFS)
	require.NoError(t, err)

	languages := service.GetSupportedLanguages()
	assert.Len(t, languages, 3)
	assert.Contains(t, languages, "en")
	assert.Contains(t, languages, "uk")
	assert.Contains(t, languages, "de")
}

func TestLocalizationService_GetLanguageByCode(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	service := NewLocalizationService(logger)

	mockFS := fstest.MapFS{
		"languages.json": &fstest.MapFile{
			Data: []byte(`{
				"en": {"code": "en", "name": "English", "flag": "🇬🇧"},
				"uk": {"code": "uk", "name": "Ukrainian", "flag": "🇺🇦"}
			}`),
		},
		"en.json": &fstest.MapFile{Data: []byte(`{"key": "value"}`)},
		"uk.json": &fstest.MapFile{Data: []byte(`{"key": "value"}`)},
	}

	err := service.LoadTranslations(mockFS)
	require.NoError(t, err)

	t.Run("existing language code", func(t *testing.T) {
		lang, exists := service.GetLanguageByCode("en")
		assert.True(t, exists)
		assert.Equal(t, "en", lang.Code)
		assert.Equal(t, "English", lang.Name)
		assert.Equal(t, "🇬🇧", lang.Flag)
	})

	t.Run("another existing language code", func(t *testing.T) {
		lang, exists := service.GetLanguageByCode("uk")
		assert.True(t, exists)
		assert.Equal(t, "uk", lang.Code)
		assert.Equal(t, "Ukrainian", lang.Name)
	})

	t.Run("nonexistent language code returns default", func(t *testing.T) {
		lang, exists := service.GetLanguageByCode("fr")
		assert.False(t, exists)
		assert.Equal(t, "en", lang.Code) // Should return default language
	})
}

func TestLocalizationService_DetectLanguageFromName(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	service := NewLocalizationService(logger)

	mockFS := fstest.MapFS{
		"languages.json": &fstest.MapFile{
			Data: []byte(`{
				"en": {"code": "en", "name": "English", "flag": "🇬🇧"},
				"uk": {"code": "uk", "name": "Ukrainian", "flag": "🇺🇦"}
			}`),
		},
		"en.json": &fstest.MapFile{Data: []byte(`{"key": "value"}`)},
		"uk.json": &fstest.MapFile{Data: []byte(`{"key": "value"}`)},
	}

	err := service.LoadTranslations(mockFS)
	require.NoError(t, err)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"english lowercase", "english", "en"},
		{"english capitalized", "English", "en"},
		{"ukrainian", "ukrainian", "uk"},
		{"ukrainian cyrillic", "українська", "uk"},
		{"german", "german", "de"},
		{"deutsch", "deutsch", "de"},
		{"french", "french", "fr"},
		{"français", "français", "fr"},
		{"spanish", "spanish", "es"},
		{"español", "español", "es"},
		{"language code en", "en", "en"},
		{"language code uk", "uk", "uk"},
		{"unknown language", "unknown", "en"},
		{"empty string", "", "en"},
		{"whitespace", "  ", "en"},
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
				"en": {"code": "en", "name": "English", "flag": "🇬🇧"},
				"uk": {"code": "uk", "name": "Ukrainian", "flag": "🇺🇦"}
			}`),
		},
		"en.json": &fstest.MapFile{
			Data: []byte(`{
				"greeting": "Hello",
				"weather": "Weather",
				"temperature": "Temperature"
			}`),
		},
		"uk.json": &fstest.MapFile{
			Data: []byte(`{
				"greeting": "Привіт",
				"weather": "Погода"
			}`),
		},
	}

	err := service.LoadTranslations(mockFS)
	require.NoError(t, err)

	t.Run("get keys for existing language", func(t *testing.T) {
		keys := service.GetAvailableTranslationKeys("en")
		assert.Len(t, keys, 3)
		assert.Contains(t, keys, "greeting")
		assert.Contains(t, keys, "weather")
		assert.Contains(t, keys, "temperature")
	})

	t.Run("get keys for another language", func(t *testing.T) {
		keys := service.GetAvailableTranslationKeys("uk")
		assert.Len(t, keys, 2)
		assert.Contains(t, keys, "greeting")
		assert.Contains(t, keys, "weather")
	})

	t.Run("get keys for nonexistent language", func(t *testing.T) {
		keys := service.GetAvailableTranslationKeys("fr")
		assert.Nil(t, keys)
	})
}

func TestLocalizationService_ConcurrentAccess(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	service := NewLocalizationService(logger)

	mockFS := fstest.MapFS{
		"languages.json": &fstest.MapFile{
			Data: []byte(`{
				"en": {"code": "en", "name": "English", "flag": "🇬🇧"}
			}`),
		},
		"en.json": &fstest.MapFile{
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
				_ = service.T(ctx, "en", "greeting")
				_ = service.IsLanguageSupported("en")
				_ = service.GetAvailableTranslationKeys("en")
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
				_ = service.T(ctx, "en", "greeting")
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
				"en": {"code": "en", "name": "English", "flag": "🇬🇧"}
			}`),
		},
		"en.json": &fstest.MapFile{
			Data: []byte(`{}`), // Empty translations
		},
	}

	err := service.LoadTranslations(mockFS)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("translate with empty translations", func(t *testing.T) {
		result := service.T(ctx, "en", "any_key")
		assert.Equal(t, "any_key", result) // Should return key
	})

	t.Run("get keys returns empty slice", func(t *testing.T) {
		keys := service.GetAvailableTranslationKeys("en")
		assert.Empty(t, keys)
	})
}