package service

import (
	"context"
	"os"
	"testing"

	"github.com/apppanel/durablelinks-core/models"
	"github.com/apppanel/durablelinks-core/repository"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	_ = log
	os.Exit(m.Run())
}

func TestParseLongDurableLink(t *testing.T) {
	tests := []struct {
		name     string
		longLink string
		want     models.CreateDurableLinkRequest
		wantErr  bool
	}{
		{
			name: "complete link with all parameters",
			longLink: "https://example.com?link=https://target.com" +
				"&apn=com.android.app" +
				"&afl=https://android-fallback.com" +
				"&amv=123" +
				"&isi=123456789" +
				"&ifl=https://ios-fallback.com" +
				"&ipfl=https://ipad-fallback.com" +
				"&ofl=https://other-platform-fallback.com" +
				"&utm_source=source" +
				"&utm_medium=medium" +
				"&utm_campaign=campaign" +
				"&utm_term=term" +
				"&utm_content=content" +
				"&at=at" +
				"&ct=ct" +
				"&mt=mt" +
				"&pt=pt" +
				"&st=social title" +
				"&sd=social description" +
				"&si=https://social-image.com" +
				"&path=SHORT",
			want: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://target.com",
					AndroidParameters: models.AndroidParameters{
						AndroidPackageName:           "com.android.app",
						AndroidFallbackLink:          "https://android-fallback.com",
						AndroidMinPackageVersionCode: "123",
					},
					IosParameters: models.IOSParameters{
						IOSAppStoreId:       "123456789",
						IOSFallbackLink:     "https://ios-fallback.com",
						IOSIpadFallbackLink: "https://ipad-fallback.com",
					},
					OtherPlatformParameters: models.OtherPlatformParameters{
						FallbackURL: "https://other-platform-fallback.com",
					},
					AnalyticsInfo: models.AnalyticsInfo{
						MarketingParameters: models.MarketingParameters{
							UtmSource:   "source",
							UtmMedium:   "medium",
							UtmCampaign: "campaign",
							UtmTerm:     "term",
							UtmContent:  "content",
						},
						ItunesConnectAnalytics: models.ITunesConnectAnalytics{
							At: "at",
							Ct: "ct",
							Mt: "mt",
							Pt: "pt",
						},
					},
					SocialMetaTagInfo: models.SocialMetaTagInfo{
						SocialTitle:       "social title",
						SocialDescription: "social description",
						SocialImageLink:   "https://social-image.com",
					},
				},
				Suffix: models.Suffix{
					Option: "SHORT",
				},
			},
			wantErr: false,
		},
		{
			name:     "invalid URL format",
			longLink: "not a valid url",
			want:     models.CreateDurableLinkRequest{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &linkService{}
			got, err := service.parseLongDurableLink(tt.longLink)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCreateDurableLink_Warnings(t *testing.T) {
	tests := []struct {
		name             string
		params           models.CreateDurableLinkRequest
		expectedWarnings []models.Warning
	}{
		{
			name: "mt param without pt should warn",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
					IosParameters: models.IOSParameters{
						IOSAppStoreId: "123456789",
					},
					AnalyticsInfo: models.AnalyticsInfo{
						ItunesConnectAnalytics: models.ITunesConnectAnalytics{
							Mt: "8",
						},
					},
				},
			},
			expectedWarnings: []models.Warning{
				{
					WarningCode:    "UNRECOGNIZED_PARAM",
					WarningMessage: "Param 'mt' is not needed, since 'pt' is not specified.",
				},
			},
		},
		{
			name: "at param without pt should warn",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
					IosParameters: models.IOSParameters{
						IOSAppStoreId: "123456789",
					},
					AnalyticsInfo: models.AnalyticsInfo{
						ItunesConnectAnalytics: models.ITunesConnectAnalytics{
							At: "affiliate_token",
							// Pt is empty - should trigger warning
						},
					},
				},
			},
			expectedWarnings: []models.Warning{
				{
					WarningCode:    "UNRECOGNIZED_PARAM",
					WarningMessage: "Param 'at' is not needed, since 'pt' is not specified.",
				},
			},
		},
		{
			name: "ct param without pt should warn",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
					IosParameters: models.IOSParameters{
						IOSAppStoreId: "123456789",
					},
					AnalyticsInfo: models.AnalyticsInfo{
						ItunesConnectAnalytics: models.ITunesConnectAnalytics{
							Ct: "campaign_token",
							// Pt is empty - should trigger warning
						},
					},
				},
			},
			expectedWarnings: []models.Warning{
				{
					WarningCode:    "UNRECOGNIZED_PARAM",
					WarningMessage: "Param 'ct' is not needed, since 'pt' is not specified.",
				},
			},
		},
		{
			name: "pt param without isi should warn",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
					AnalyticsInfo: models.AnalyticsInfo{
						ItunesConnectAnalytics: models.ITunesConnectAnalytics{
							Pt: "provider_token",
						},
					},
				},
			},
			expectedWarnings: []models.Warning{
				{
					WarningCode:    "UNRECOGNIZED_PARAM",
					WarningMessage: "Param 'pt' is not needed, since 'isi' is not specified.",
				},
			},
		},
		{
			name: "invalid social image URL should warn",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
					SocialMetaTagInfo: models.SocialMetaTagInfo{
						SocialImageLink: "not-a-valid-url",
					},
				},
			},
			expectedWarnings: []models.Warning{
				{
					WarningCode:    "MALFORMED_PARAM",
					WarningMessage: "Param 'si' is not a valid URL",
				},
			},
		},
		{
			name: "valid params should have no warnings",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
					IosParameters: models.IOSParameters{
						IOSAppStoreId: "123456789",
					},
					AnalyticsInfo: models.AnalyticsInfo{
						ItunesConnectAnalytics: models.ITunesConnectAnalytics{
							Pt: "provider_token",
							At: "affiliate_token",
							Ct: "campaign_token",
							Mt: "8",
						},
					},
					SocialMetaTagInfo: models.SocialMetaTagInfo{
						SocialImageLink: "https://example.com/image.png",
					},
				},
			},
			expectedWarnings: []models.Warning{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			repo := repository.NewLinkRepository(db)
			service := NewLinkService(repo)
			tenantCfg := TenantConfig{
				URLScheme:             "https",
				DomainAllowList:       []string{"example.com"},
				ShortPathLength:       8,
				UnguessablePathLength: 17,
			}

			mock.ExpectExec(`INSERT INTO apppanel_durable_links`).
				WillReturnResult(sqlmock.NewResult(1, 1))

			result, err := service.CreateDurableLink(context.Background(), tt.params, nil, tenantCfg)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, len(tt.expectedWarnings), len(result.Warnings),
				"Expected %d warnings but got %d", len(tt.expectedWarnings), len(result.Warnings))

			for i, expectedWarning := range tt.expectedWarnings {
				if i < len(result.Warnings) {
					assert.Equal(t, expectedWarning.WarningCode, result.Warnings[i].WarningCode)
					assert.Equal(t, expectedWarning.WarningMessage, result.Warnings[i].WarningMessage)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGenerateDurableLinkPath(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{
			name:   "short path",
			length: 6,
		},
		{
			name:   "medium path",
			length: 12,
		},
		{
			name:   "long path",
			length: 24,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := make(map[string]bool)
			const iterations = 1000

			for range iterations {
				path := generateDurableLinkPath(tt.length)

				// Test length
				if len(path) != tt.length {
					t.Errorf("generateDurableLinkPath(%d) length = %d, want %d", tt.length, len(path), tt.length)
				}

				// Test character set - must be alphanumeric
				for _, r := range path {
					if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
						t.Errorf("generateDurableLinkPath(%d) contains invalid character: %c", tt.length, r)
					}
				}

				// Test uniqueness
				if paths[path] {
					t.Errorf("generateDurableLinkPath(%d) generated duplicate path: %s", tt.length, path)
				}
				paths[path] = true
			}

			// Test distribution - should generate both letters and numbers
			hasLetters := false
			hasNumbers := false
			for path := range paths {
				for _, r := range path {
					if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
						hasLetters = true
					}
					if r >= '0' && r <= '9' {
						hasNumbers = true
					}
				}
			}

			if !hasLetters || !hasNumbers {
				t.Errorf("generateDurableLinkPath(%d) does not generate both letters and numbers", tt.length)
			}
		})
	}
}

func TestRemovePreviewFromHost(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "prefix preview",
			input:    "preview.acme.short.link",
			expected: "acme.short.link",
		},
		{
			name:     "hyphenated preview",
			input:    "acme-preview.short.link",
			expected: "acme.short.link",
		},
		{
			name:     "no preview",
			input:    "acme.short.link",
			expected: "acme.short.link",
		},
		{
			name:     "prefix preview with nested domain",
			input:    "preview.staging.acme.short.link",
			expected: "staging.acme.short.link",
		},
		{
			name:     "hyphenated preview with dot in domain",
			input:    "myapp-preview.dev.short.link",
			expected: "myapp.dev.short.link",
		},
		{
			name:     "unrelated prefix",
			input:    "notpreview.acme.short.link",
			expected: "notpreview.acme.short.link",
		},
		{
			name:     "unrelated suffix",
			input:    "acme-somethingelse.short.link",
			expected: "acme-somethingelse.short.link",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := removePreviewFromHost(tc.input)
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}
