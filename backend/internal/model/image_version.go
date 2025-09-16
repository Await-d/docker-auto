package model

import (
	"time"

	"gorm.io/gorm"
)

// ImageVersion represents cached image version information
type ImageVersion struct {
	ID           int       `json:"id" gorm:"primaryKey;autoIncrement"`
	ImageName    string    `json:"image_name" gorm:"not null;size:255;uniqueIndex:unique_image_tag_registry"`
	Tag          string    `json:"tag" gorm:"not null;size:100;uniqueIndex:unique_image_tag_registry;index:idx_image_versions_image_tag"`
	Digest       string    `json:"digest" gorm:"not null;size:71;index:idx_image_versions_digest"`
	SizeBytes    int64     `json:"size_bytes" gorm:"default:0"`
	PublishedAt  *time.Time `json:"published_at,omitempty" gorm:"index:idx_image_versions_published_at"`
	Architecture string    `json:"architecture" gorm:"size:50;default:'amd64'"`
	OS           string    `json:"os" gorm:"size:50;default:'linux'"`
	LayersInfo   string    `json:"layers_info,omitempty" gorm:"type:jsonb;default:'[]'"`
	RegistryURL  string    `json:"registry_url" gorm:"size:255;default:'docker.io';uniqueIndex:unique_image_tag_registry"`
	CheckedAt    time.Time `json:"checked_at" gorm:"index:idx_image_versions_checked_at"`
	IsLatest     bool      `json:"is_latest" gorm:"not null;default:false;index:idx_image_versions_is_latest"`
	Metadata     string    `json:"metadata,omitempty" gorm:"type:jsonb;default:'{}'"`
}

// ImageVersionFilter represents filters for querying image versions
type ImageVersionFilter struct {
	ImageName    string    `json:"image_name,omitempty"`
	Tag          string    `json:"tag,omitempty"`
	RegistryURL  string    `json:"registry_url,omitempty"`
	Architecture string    `json:"architecture,omitempty"`
	OS           string    `json:"os,omitempty"`
	IsLatest     *bool     `json:"is_latest,omitempty"`
	CheckedAfter *time.Time `json:"checked_after,omitempty"`
	CheckedBefore *time.Time `json:"checked_before,omitempty"`
	PublishedAfter *time.Time `json:"published_after,omitempty"`
	PublishedBefore *time.Time `json:"published_before,omitempty"`
	Limit        int       `json:"limit,omitempty"`
	Offset       int       `json:"offset,omitempty"`
	OrderBy      string    `json:"order_by,omitempty"`
}

// ImageSummary represents a summary of image information
type ImageSummary struct {
	ImageName     string    `json:"image_name"`
	RegistryURL   string    `json:"registry_url"`
	LatestTag     string    `json:"latest_tag"`
	LatestDigest  string    `json:"latest_digest"`
	TagCount      int       `json:"tag_count"`
	LastChecked   time.Time `json:"last_checked"`
	LastPublished *time.Time `json:"last_published,omitempty"`
}

// TableName returns the table name for ImageVersion model
func (ImageVersion) TableName() string {
	return "image_versions"
}

// GetFullImageName returns full image name with registry
func (iv *ImageVersion) GetFullImageName() string {
	if iv.RegistryURL == "" || iv.RegistryURL == "docker.io" {
		return iv.ImageName
	}
	return iv.RegistryURL + "/" + iv.ImageName
}

// GetFullImageNameWithTag returns full image name with registry and tag
func (iv *ImageVersion) GetFullImageNameWithTag() string {
	fullName := iv.GetFullImageName()
	if iv.Tag == "" {
		return fullName + ":latest"
	}
	return fullName + ":" + iv.Tag
}

// IsStale checks if the image version check is stale (older than specified duration)
func (iv *ImageVersion) IsStale(staleDuration time.Duration) bool {
	return time.Since(iv.CheckedAt) > staleDuration
}

// HasSize checks if the image has size information
func (iv *ImageVersion) HasSize() bool {
	return iv.SizeBytes > 0
}

// GetSizeInMB returns the size in megabytes
func (iv *ImageVersion) GetSizeInMB() float64 {
	return float64(iv.SizeBytes) / (1024 * 1024)
}

// GetSizeInGB returns the size in gigabytes
func (iv *ImageVersion) GetSizeInGB() float64 {
	return float64(iv.SizeBytes) / (1024 * 1024 * 1024)
}

// IsNewerThan checks if this version is newer than another version
func (iv *ImageVersion) IsNewerThan(other *ImageVersion) bool {
	if iv.PublishedAt == nil || other.PublishedAt == nil {
		return iv.CheckedAt.After(other.CheckedAt)
	}
	return iv.PublishedAt.After(*other.PublishedAt)
}

// GetArchitectures returns common architecture values
func GetCommonArchitectures() []string {
	return []string{
		"amd64",
		"arm64",
		"arm/v7",
		"arm/v6",
		"386",
		"ppc64le",
		"s390x",
	}
}

// GetOperatingSystems returns common OS values
func GetCommonOperatingSystems() []string {
	return []string{
		"linux",
		"windows",
		"darwin",
	}
}

// BeforeCreate hook for ImageVersion model
func (iv *ImageVersion) BeforeCreate(tx *gorm.DB) error {
	if iv.Tag == "" {
		iv.Tag = "latest"
	}
	if iv.Architecture == "" {
		iv.Architecture = "amd64"
	}
	if iv.OS == "" {
		iv.OS = "linux"
	}
	if iv.RegistryURL == "" {
		iv.RegistryURL = "docker.io"
	}
	if iv.CheckedAt.IsZero() {
		iv.CheckedAt = time.Now()
	}
	return nil
}

// BeforeUpdate hook for ImageVersion model
func (iv *ImageVersion) BeforeUpdate(tx *gorm.DB) error {
	// Update checked time on any update
	iv.CheckedAt = time.Now()
	return nil
}

// BeforeSave hook for ImageVersion model
func (iv *ImageVersion) BeforeSave(tx *gorm.DB) error {
	// If this is marked as latest, unmark other versions of the same image
	if iv.IsLatest {
		err := tx.Model(&ImageVersion{}).
			Where("image_name = ? AND registry_url = ? AND id != ?", iv.ImageName, iv.RegistryURL, iv.ID).
			Update("is_latest", false).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// CleanupStaleVersions removes old image versions to maintain cache size
func CleanupStaleVersions(tx *gorm.DB, retentionDays int, maxVersionsPerImage int) error {
	// Delete versions older than retention period
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	err := tx.Where("checked_at < ? AND is_latest = false", cutoffDate).Delete(&ImageVersion{}).Error
	if err != nil {
		return err
	}

	// Keep only the latest N versions per image
	var images []struct {
		ImageName   string
		RegistryURL string
	}

	err = tx.Model(&ImageVersion{}).
		Select("DISTINCT image_name, registry_url").
		Find(&images).Error
	if err != nil {
		return err
	}

	for _, img := range images {
		var versions []ImageVersion
		err = tx.Where("image_name = ? AND registry_url = ?", img.ImageName, img.RegistryURL).
			Order("checked_at DESC").
			Offset(maxVersionsPerImage).
			Find(&versions).Error
		if err != nil {
			continue
		}

		for _, v := range versions {
			if !v.IsLatest {
				tx.Delete(&v)
			}
		}
	}

	return nil
}