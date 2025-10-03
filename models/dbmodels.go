package models

import (
	"crypto/sha256"
	"fmt"
	"time"
)

type DurableLinkDB struct {
	ID                  int64     `db:"id"`
	Host                string    `db:"host"`
	Path                string    `db:"path"`
	Link                string    `db:"link"`
	IsUnguessablePath   bool      `db:"is_unguessable_path"`
	ProjectID           *string   `db:"project_id"`
	AndroidPackageName  *string   `db:"android_package_name"`
	AndroidFallbackLink *string   `db:"android_fallback_link"`
	AndroidMinVersion   *string   `db:"android_min_version"`
	IOSFallbackLink     *string   `db:"ios_fallback_link"`
	IOSIpadFallbackLink *string   `db:"ios_ipad_fallback_link"`
	IOSAppStoreID       *int64    `db:"ios_app_store_id"`
	SocialTitle         *string   `db:"social_title"`
	SocialDescription   *string   `db:"social_description"`
	SocialImageLink     *string   `db:"social_image_link"`
	UtmSource           *string   `db:"utm_source"`
	UtmMedium           *string   `db:"utm_medium"`
	UtmCampaign         *string   `db:"utm_campaign"`
	UtmTerm             *string   `db:"utm_term"`
	UtmContent          *string   `db:"utm_content"`
	ItunesPt            *string   `db:"itunes_pt"`
	ItunesAt            *string   `db:"itunes_at"`
	ItunesCt            *string   `db:"itunes_ct"`
	ItunesMt            *string   `db:"itunes_mt"`
	OtherFallbackURL    *string   `db:"other_fallback_url"`
	ParamsHash          string    `db:"params_hash"`
	CreatedAt           time.Time `db:"created_at"`
	UpdatedAt           time.Time `db:"updated_at"`
}

func (db *DurableLinkDB) ToDurableLink() DurableLink {
	return DurableLink{
		Host: db.Host,
		Link: db.Link,
		AndroidParameters: AndroidParameters{
			AndroidPackageName:           db.AndroidPackageName,
			AndroidFallbackLink:          db.AndroidFallbackLink,
			AndroidMinPackageVersionCode: db.AndroidMinVersion,
		},
		IosParameters: IOSParameters{
			IOSFallbackLink:     db.IOSFallbackLink,
			IOSIpadFallbackLink: db.IOSIpadFallbackLink,
			IOSAppStoreId:       db.IOSAppStoreID,
		},
		OtherPlatformParameters: OtherPlatformParameters{
			FallbackURL: db.OtherFallbackURL,
		},
		SocialMetaTagInfo: SocialMetaTagInfo{
			SocialTitle:       db.SocialTitle,
			SocialDescription: db.SocialDescription,
			SocialImageLink:   db.SocialImageLink,
		},
		AnalyticsInfo: AnalyticsInfo{
			MarketingParameters: MarketingParameters{
				UtmSource:   db.UtmSource,
				UtmMedium:   db.UtmMedium,
				UtmCampaign: db.UtmCampaign,
				UtmTerm:     db.UtmTerm,
				UtmContent:  db.UtmContent,
			},
			ItunesConnectAnalytics: ITunesConnectAnalytics{
				Pt: db.ItunesPt,
				At: db.ItunesAt,
				Ct: db.ItunesCt,
				Mt: db.ItunesMt,
			},
		},
	}
}

func FromDurableLink(dl DurableLink, host, path string, isUnguessable bool, projectID *string) *DurableLinkDB {
	db := &DurableLinkDB{
		Host:                host,
		Path:                path,
		Link:                dl.Link,
		IsUnguessablePath:   isUnguessable,
		ProjectID:           projectID,
		AndroidPackageName:  dl.AndroidParameters.AndroidPackageName,
		AndroidFallbackLink: dl.AndroidParameters.AndroidFallbackLink,
		AndroidMinVersion:   dl.AndroidParameters.AndroidMinPackageVersionCode,
		IOSFallbackLink:     dl.IosParameters.IOSFallbackLink,
		IOSIpadFallbackLink: dl.IosParameters.IOSIpadFallbackLink,
		IOSAppStoreID:       dl.IosParameters.IOSAppStoreId,
		SocialTitle:         dl.SocialMetaTagInfo.SocialTitle,
		SocialDescription:   dl.SocialMetaTagInfo.SocialDescription,
		SocialImageLink:     dl.SocialMetaTagInfo.SocialImageLink,
		UtmSource:           dl.AnalyticsInfo.MarketingParameters.UtmSource,
		UtmMedium:           dl.AnalyticsInfo.MarketingParameters.UtmMedium,
		UtmCampaign:         dl.AnalyticsInfo.MarketingParameters.UtmCampaign,
		UtmTerm:             dl.AnalyticsInfo.MarketingParameters.UtmTerm,
		UtmContent:          dl.AnalyticsInfo.MarketingParameters.UtmContent,
		ItunesPt:            dl.AnalyticsInfo.ItunesConnectAnalytics.Pt,
		ItunesAt:            dl.AnalyticsInfo.ItunesConnectAnalytics.At,
		ItunesCt:            dl.AnalyticsInfo.ItunesConnectAnalytics.Ct,
		ItunesMt:            dl.AnalyticsInfo.ItunesConnectAnalytics.Mt,
		OtherFallbackURL:    dl.OtherPlatformParameters.FallbackURL,
	}
	db.ParamsHash = db.ComputeParamsHash()
	return db
}

// ComputeParamsHash computes a SHA256 hash of all optional parameters for efficient duplicate detection
func (db *DurableLinkDB) ComputeParamsHash() string {
	// Build a deterministic string representation of all optional parameters
	var parts []string

	parts = append(parts, stringPtrOrEmpty(db.AndroidPackageName))
	parts = append(parts, stringPtrOrEmpty(db.AndroidFallbackLink))
	parts = append(parts, stringPtrOrEmpty(db.AndroidMinVersion))
	parts = append(parts, stringPtrOrEmpty(db.IOSFallbackLink))
	parts = append(parts, stringPtrOrEmpty(db.IOSIpadFallbackLink))
	parts = append(parts, int64PtrToString(db.IOSAppStoreID))
	parts = append(parts, stringPtrOrEmpty(db.SocialTitle))
	parts = append(parts, stringPtrOrEmpty(db.SocialDescription))
	parts = append(parts, stringPtrOrEmpty(db.SocialImageLink))
	parts = append(parts, stringPtrOrEmpty(db.UtmSource))
	parts = append(parts, stringPtrOrEmpty(db.UtmMedium))
	parts = append(parts, stringPtrOrEmpty(db.UtmCampaign))
	parts = append(parts, stringPtrOrEmpty(db.UtmTerm))
	parts = append(parts, stringPtrOrEmpty(db.UtmContent))
	parts = append(parts, stringPtrOrEmpty(db.ItunesPt))
	parts = append(parts, stringPtrOrEmpty(db.ItunesAt))
	parts = append(parts, stringPtrOrEmpty(db.ItunesCt))
	parts = append(parts, stringPtrOrEmpty(db.ItunesMt))
	parts = append(parts, stringPtrOrEmpty(db.OtherFallbackURL))
	combined := ""
	for i, part := range parts {
		if i > 0 {
			combined += "\x00"
		}
		combined += part
	}
	hash := sha256.Sum256([]byte(combined))
	return fmt.Sprintf("%x", hash)
}

func stringPtrOrEmpty(s *string) string {
	if s == nil {
		return "\x01"
	}
	return *s
}

func int64PtrToString(i *int64) string {
	if i == nil {
		return "\x01"
	}
	return fmt.Sprintf("%d", *i)
}
