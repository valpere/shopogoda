package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"github.com/valpere/shopogoda/internal"
)

// SupportedLanguage represents a supported language
type SupportedLanguage struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Flag string `json:"flag"`
}

type SupportedLanguages map[string]SupportedLanguage

type Translation map[string]string

type Translations map[string]Translation

// LocalizationService handles multi-language support
type LocalizationService struct {
	translations       Translations       // [language][key] = translation
	supportedLanguages SupportedLanguages // Supported languages
	defaultLanguage    string             // fallback language (English)
	logger             *zerolog.Logger
	mu                 sync.RWMutex
}

// NewLocalizationService creates a new localization service
func NewLocalizationService(logger *zerolog.Logger) *LocalizationService {
	return &LocalizationService{
		translations:    make(Translations),
		defaultLanguage: internal.DefaultLanguage,
		logger:          logger,
	}
}

// LoadTranslations loads translation files from embedded filesystem
func (ls *LocalizationService) LoadTranslations(localesFS fs.FS) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	data, err := fs.ReadFile(localesFS, "languages.json")
	if err != nil {
		return err
	}

	ls.supportedLanguages = make(SupportedLanguages)
	if err := json.Unmarshal(data, &ls.supportedLanguages); err != nil {
		return err
	}

	for code := range ls.supportedLanguages {
		filename := fmt.Sprintf("%s.json", code)

		data, err := fs.ReadFile(localesFS, filename)
		if err != nil {
			ls.logger.Error().
				Err(err).
				Str("language", code).
				Str("file", filename).
				Msg("Failed to read translation file")
			continue
		}

		translations := make(Translation)
		if err := json.Unmarshal(data, &translations); err != nil {
			ls.logger.Error().
				Err(err).
				Str("language", code).
				Msg("Failed to parse translation file")
			continue
		}

		ls.translations[code] = translations
		ls.logger.Info().
			Str("language", code).
			Int("keys", len(translations)).
			Msg("Loaded translations")
	}

	return nil
}

// T translates a key to the specified language
func (ls *LocalizationService) T(ctx context.Context, language, key string, args ...any) string {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	// Try to get translation in requested language
	if langMap, exists := ls.translations[language]; exists {
		if translation, exists := langMap[key]; exists {
			if len(args) > 0 {
				return fmt.Sprintf(translation, args...)
			}
			return translation
		}
	}

	// Fall back to default language
	if langMap, exists := ls.translations[ls.defaultLanguage]; exists {
		if translation, exists := langMap[key]; exists {
			ls.logger.Debug().
				Str("key", key).
				Str("requested_lang", language).
				Str("fallback_lang", ls.defaultLanguage).
				Msg("Using fallback language for translation")

			if len(args) > 0 {
				return fmt.Sprintf(translation, args...)
			}
			return translation
		}
	}

	// If no translation found, return the key itself
	ls.logger.Warn().
		Str("key", key).
		Str("language", language).
		Msg("Translation key not found")

	return key
}

// IsLanguageSupported checks if a language code is supported
func (ls *LocalizationService) IsLanguageSupported(language string) bool {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	_, exists := ls.translations[language]
	return exists
}

// GetSupportedLanguages returns all supported languages
func (ls *LocalizationService) GetSupportedLanguages() SupportedLanguages {
	return ls.supportedLanguages
}

// GetLanguageByCode returns language info by code
func (ls *LocalizationService) GetLanguageByCode(code string) (SupportedLanguage, bool) {
	if lang, exists := ls.supportedLanguages[code]; exists {
		return lang, exists
	}
	return ls.supportedLanguages[internal.DefaultLanguage], false
}

// DetectLanguageFromName tries to detect language from a name/description
func (ls *LocalizationService) DetectLanguageFromName(name string) string {
	originalName := strings.TrimSpace(name)
	lowerName := strings.ToLower(originalName)

	// Map common language names to full IETF codes
	nameMap := map[string]string{
		"english":    "en-US",
		"ukrainian":  "uk-UA",
		"українська": "uk-UA",
		"deutsch":    "de-DE",
		"german":     "de-DE",
		"français":   "fr-FR",
		"french":     "fr-FR",
		"español":    "es-ES",
		"spanish":    "es-ES",
	}

	if code, exists := nameMap[lowerName]; exists {
		return code
	}

	// Check if it's already a valid code (preserving original case)
	if ls.IsLanguageSupported(originalName) {
		return originalName
	}

	return ls.defaultLanguage
}

// GetAvailableTranslationKeys returns all available translation keys for a language
func (ls *LocalizationService) GetAvailableTranslationKeys(language string) []string {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	langMap, exists := ls.translations[language]
	if !exists {
		return nil
	}

	keys := make([]string, 0, len(langMap))
	for key := range langMap {
		keys = append(keys, key)
	}

	return keys
}
