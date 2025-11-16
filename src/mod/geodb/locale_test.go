package geodb_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"imuslab.com/zoraxy/mod/geodb"
	"imuslab.com/zoraxy/mod/info/logger"
)

func TestGetLocaleFromCountryCode(t *testing.T) {
	tests := []struct {
		name         string
		countryCode  string
		expectedLang string
	}{
		{
			name:         "Arabic - AA",
			countryCode:  "aa",
			expectedLang: "ar_AA",
		},
		{
			name:         "Belarusian - BY",
			countryCode:  "by",
			expectedLang: "be_BY",
		},
		{
			name:         "Bulgarian - BG",
			countryCode:  "bg",
			expectedLang: "bg_BG",
		},
		{
			name:         "Catalan - ES",
			countryCode:  "es",
			expectedLang: "ca_ES",
		},
		{
			name:         "Czech - CZ",
			countryCode:  "cz",
			expectedLang: "cs_CZ",
		},
		{
			name:         "Danish - DK",
			countryCode:  "dk",
			expectedLang: "da_DK",
		},
		{
			name:         "German Switzerland - CH",
			countryCode:  "ch",
			expectedLang: "de_CH",
		},
		{
			name:         "German - DE",
			countryCode:  "de",
			expectedLang: "de_DE",
		},
		{
			name:         "Greek - GR",
			countryCode:  "gr",
			expectedLang: "el_GR",
		},
		{
			name:         "English Australia - AU",
			countryCode:  "au",
			expectedLang: "en_AU",
		},
		{
			name:         "English Belgium - BE",
			countryCode:  "be",
			expectedLang: "en_BE",
		},
		{
			name:         "English Great Britain - GB",
			countryCode:  "gb",
			expectedLang: "en_GB",
		},
		{
			name:         "English Japan - JP",
			countryCode:  "jp",
			expectedLang: "en_JP",
		},
		{
			name:         "English United States - US",
			countryCode:  "us",
			expectedLang: "en_US",
		},
		{
			name:         "English South Africa - ZA",
			countryCode:  "za",
			expectedLang: "en_ZA",
		},
		{
			name:         "Finnish - FI",
			countryCode:  "fi",
			expectedLang: "fi_FI",
		},
		{
			name:         "French Canada - CA",
			countryCode:  "ca",
			expectedLang: "fr_CA",
		},
		{
			name:         "French - FR",
			countryCode:  "fr",
			expectedLang: "fr_FR",
		},
		{
			name:         "Croatian - HR",
			countryCode:  "hr",
			expectedLang: "hr_HR",
		},
		{
			name:         "Hungarian - HU",
			countryCode:  "hu",
			expectedLang: "hu_HU",
		},
		{
			name:         "Icelandic - IS",
			countryCode:  "is",
			expectedLang: "is_IS",
		},
		{
			name:         "Italian - IT",
			countryCode:  "it",
			expectedLang: "it_IT",
		},
		{
			name:         "Hebrew - IL",
			countryCode:  "il",
			expectedLang: "iw_IL",
		},
		{
			name:         "Korean - KR",
			countryCode:  "kr",
			expectedLang: "ko_KR",
		},
		{
			name:         "Lithuanian - LT",
			countryCode:  "lt",
			expectedLang: "lt_LT",
		},
		{
			name:         "Latvian - LV",
			countryCode:  "lv",
			expectedLang: "lv_LV",
		},
		{
			name:         "Macedonian - MK",
			countryCode:  "mk",
			expectedLang: "mk_MK",
		},
		{
			name:         "Dutch - NL",
			countryCode:  "nl",
			expectedLang: "nl_NL",
		},
		{
			name:         "Norwegian - NO",
			countryCode:  "no",
			expectedLang: "no_NO",
		},
		{
			name:         "Polish - PL",
			countryCode:  "pl",
			expectedLang: "pl_PL",
		},
		{
			name:         "Portuguese Brazil - BR",
			countryCode:  "br",
			expectedLang: "pt_BR",
		},
		{
			name:         "Portuguese - PT",
			countryCode:  "pt",
			expectedLang: "pt_PT",
		},
		{
			name:         "Romanian - RO",
			countryCode:  "ro",
			expectedLang: "ro_RO",
		},
		{
			name:         "Russian - RU",
			countryCode:  "ru",
			expectedLang: "ru_RU",
		},
		{
			name:         "Serbian - SP",
			countryCode:  "sp",
			expectedLang: "sh_SP",
		},
		{
			name:         "Slovak - SK",
			countryCode:  "sk",
			expectedLang: "sk_SK",
		},
		{
			name:         "Slovenian - SL",
			countryCode:  "sl",
			expectedLang: "sl_SL",
		},
		{
			name:         "Albanian - AL",
			countryCode:  "al",
			expectedLang: "sq_AL",
		},
		{
			name:         "Swedish - SE",
			countryCode:  "se",
			expectedLang: "sv_SE",
		},
		{
			name:         "Thai - TH",
			countryCode:  "th",
			expectedLang: "th_TH",
		},
		{
			name:         "Turkish - TR",
			countryCode:  "tr",
			expectedLang: "tr_TR",
		},
		{
			name:         "Ukrainian - UA",
			countryCode:  "ua",
			expectedLang: "uk_UA",
		},
		{
			name:         "Chinese China - CN",
			countryCode:  "cn",
			expectedLang: "zh_CN",
		},
		{
			name:         "Chinese Taiwan - TW",
			countryCode:  "tw",
			expectedLang: "zh_TW",
		},
		{
			name:         "Chinese Hong Kong - HK",
			countryCode:  "hk",
			expectedLang: "zh_HK",
		},
		{
			name:         "Unknown country code - defaults to en-US",
			countryCode:  "zz",
			expectedLang: "en-US",
		},
		{
			name:         "Empty country code - defaults to en-US",
			countryCode:  "",
			expectedLang: "en-US",
		},
		{
			name:         "Invalid country code - defaults to en-US",
			countryCode:  "invalid",
			expectedLang: "en-US",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := geodb.GetLocaleFromCountryCode(tt.countryCode)
			assert.Equal(t, tt.expectedLang, result)
		})
	}
}

func TestGetLocaleFromCountryCode_CaseInsensitivity(t *testing.T) {
	// Test that lowercase country codes work
	tests := []struct {
		name         string
		countryCode  string
		expectedLang string
	}{
		{
			name:         "Uppercase US",
			countryCode:  "US",
			expectedLang: "en-US", // Should return default as map expects lowercase
		},
		{
			name:         "Lowercase us",
			countryCode:  "us",
			expectedLang: "en_US",
		},
		{
			name:         "Uppercase FR",
			countryCode:  "FR",
			expectedLang: "en-US", // Should return default as map expects lowercase
		},
		{
			name:         "Lowercase fr",
			countryCode:  "fr",
			expectedLang: "fr_FR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := geodb.GetLocaleFromCountryCode(tt.countryCode)
			assert.Equal(t, tt.expectedLang, result)
		})
	}
}

func TestStore_GetLocaleFromRequest(t *testing.T) {
	// Create a test store
	store, err := geodb.NewGeoDb(nil, &geodb.StoreOptions{
		AllowSlowIpv4LookUp:          true,
		AllowSlowIpv6Lookup:          true,
		Logger:                       &logger.Logger{},
		SlowLookupCacheClearInterval: 0,
	})
	assert.NoError(t, err)
	defer store.Close()

	tests := []struct {
		name           string
		setupRequest   func() *http.Request
		expectedLocale string
	}{
		{
			name: "Private IP should return default en-US",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)
				req.RemoteAddr = "192.168.1.1:12345"
				return req
			},
			expectedLocale: "en-US",
		},
		{
			name: "Localhost should return default en-US",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)
				req.RemoteAddr = "127.0.0.1:12345"
				return req
			},
			expectedLocale: "en-US",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			locale, err := store.GetLocaleFromRequest(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedLocale, locale)
		})
	}
}
