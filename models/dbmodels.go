package models

import (
	"crypto/sha256"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type DurableLinkDB struct {
	ID                  int64      `gorm:"primaryKey;autoIncrement"`
	Host                string     `gorm:"type:varchar(255);not null;index:idx_host_path,unique,composite:host_path"`
	Path                string     `gorm:"type:varchar(255);not null;index:idx_host_path,unique,composite:host_path"`
	Link                string     `gorm:"type:text;not null"`
	IsUnguessablePath   bool       `gorm:"default:false;not null;index:idx_find_existing"`
	ProjectID           *string    `gorm:"type:uuid;index:idx_project_id"`
	AndroidPackageName  *string    `gorm:"type:varchar(255)"`
	AndroidFallbackLink *string    `gorm:"type:text"`
	AndroidMinVersion   *string    `gorm:"type:varchar(50)"`
	IOSFallbackLink     *string    `gorm:"type:text"`
	IOSIpadFallbackLink *string    `gorm:"type:text"`
	IOSAppStoreID       *int64     `gorm:"type:bigint"`
	SocialTitle         *string    `gorm:"type:varchar(500)"`
	SocialDescription   *string    `gorm:"type:text"`
	SocialImageLink     *string    `gorm:"type:text"`
	UtmSource           *string    `gorm:"type:varchar(255)"`
	UtmMedium           *string    `gorm:"type:varchar(255)"`
	UtmCampaign         *string    `gorm:"type:varchar(255)"`
	UtmTerm             *string    `gorm:"type:varchar(255)"`
	UtmContent          *string    `gorm:"type:varchar(255)"`
	ItunesPt            *string    `gorm:"type:varchar(255)"`
	ItunesAt            *string    `gorm:"type:varchar(255)"`
	ItunesCt            *string    `gorm:"type:varchar(255)"`
	ItunesMt            *string    `gorm:"type:varchar(50)"`
	OtherFallbackURL    *string    `gorm:"type:text"`
	ParamsHash          string     `gorm:"type:varchar(64);index:idx_find_existing"`
	CreatedAt           time.Time  `gorm:"autoCreateTime"`
	UpdatedAt           time.Time  `gorm:"autoUpdateTime"`
}

func (DurableLinkDB) TableName() string {
	return "apppanel_durable_links"
}

// BeforeCreate is a GORM hook that runs before creating a record
func (db *DurableLinkDB) BeforeCreate(tx *gorm.DB) error {
	db.ParamsHash = db.ComputeParamsHash()
	return nil
}

// BeforeUpdate is a GORM hook that runs before updating a record
func (db *DurableLinkDB) BeforeUpdate(tx *gorm.DB) error {
	db.ParamsHash = db.ComputeParamsHash()
	return nil
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
	return &DurableLinkDB{
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
		// ParamsHash will be auto-computed by BeforeCreate/BeforeUpdate hooks
	}
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
