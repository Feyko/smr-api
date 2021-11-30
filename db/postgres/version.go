package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/satisfactorymodding/smr-api/models"
	"github.com/satisfactorymodding/smr-api/util"

	"github.com/patrickmn/go-cache"
)

func GetVersionsByID(versionIds []string, ctx *context.Context) []Version {
	cacheKey := "GetVersionsById_" + strings.Join(versionIds, ":")
	if versions, ok := dbCache.Get(cacheKey); ok {
		return versions.([]Version)
	}

	var versions []Version
	DBCtx(ctx).Find(&versions, "id in (?)", versionIds)

	if len(versionIds) != len(versions) {
		return nil
	}

	dbCache.Set(cacheKey, versions, cache.DefaultExpiration)

	return versions
}

func GetModLatestVersions(modID string, unapproved bool, ctx *context.Context) *[]Version {
	cacheKey := "GetModLatestVersions_" + modID + "_" + fmt.Sprint(unapproved)
	if versions, ok := dbCache.Get(cacheKey); ok {
		return versions.(*[]Version)
	}

	var versions []Version

	DBCtx(ctx).Select("distinct on (mod_id, stability) *").
		Where("mod_id = ?", modID).
		Where("approved = ? AND denied = ?", !unapproved, false).
		Order("mod_id, stability, created_at desc").
		Find(&versions)

	dbCache.Set(cacheKey, &versions, cache.DefaultExpiration)

	return &versions
}

func GetModsLatestVersions(modIds []string, unapproved bool, ctx *context.Context) *[]Version {
	cacheKey := "GetModsLatestVersions_" + strings.Join(modIds, ":") + "_" + fmt.Sprint(unapproved)
	if versions, ok := dbCache.Get(cacheKey); ok {
		return versions.(*[]Version)
	}

	var versions []Version

	DBCtx(ctx).Select("distinct on (mod_id, stability) *").
		Where("mod_id in (?)", modIds).
		Where("approved = ? AND denied = ?", !unapproved, false).
		Order("mod_id, stability, created_at desc").
		Find(&versions)

	dbCache.Set(cacheKey, &versions, cache.DefaultExpiration)

	return &versions
}

func GetModVersions(modID string, limit int, offset int, orderBy string, order string, unapproved bool, ctx *context.Context) []Version {
	cacheKey := "GetModVersions_" + modID + "_" + fmt.Sprint(limit) + "_" + fmt.Sprint(offset) + "_" + orderBy + "_" + order + "_" + fmt.Sprint(unapproved)
	if versions, ok := dbCache.Get(cacheKey); ok {
		return versions.([]Version)
	}

	var versions []Version
	DBCtx(ctx).Limit(limit).Offset(offset).Order(orderBy+" "+order).Where("approved = ? AND denied = ?", !unapproved, false).Find(&versions, "mod_id = ?", modID)

	dbCache.Set(cacheKey, versions, cache.DefaultExpiration)

	return versions
}

func GetModVersionsNew(modID string, filter *models.VersionFilter, unapproved bool, ctx *context.Context) []Version {
	hash, err := filter.Hash()
	cacheKey := ""
	if err == nil {
		cacheKey = "GetModVersionsNew_" + modID + "_" + hash + "_" + fmt.Sprint(unapproved)
		if versions, ok := dbCache.Get(cacheKey); ok {
			return versions.([]Version)
		}
	}

	var versions []Version
	query := DBCtx(ctx)

	if filter != nil {
		query = query.Limit(*filter.Limit).
			Offset(*filter.Offset).
			Order(string(*filter.OrderBy) + " " + string(*filter.Order))
	}

	query.Where("approved = ? AND denied = ?", !unapproved, false).Find(&versions, "mod_id = ?", modID)

	if cacheKey != "" {
		dbCache.Set(cacheKey, versions, cache.DefaultExpiration)
	}

	return versions
}

func GetModVersion(modID string, versionID string, ctx *context.Context) *Version {
	cacheKey := "GetModVersion_" + modID + "_" + versionID
	if version, ok := dbCache.Get(cacheKey); ok {
		return version.(*Version)
	}

	var version Version
	DBCtx(ctx).First(&version, "mod_id = ? AND id = ?", modID, versionID)

	if version.ID == "" {
		return nil
	}

	dbCache.Set(cacheKey, &version, cache.DefaultExpiration)

	return &version
}

func GetModVersionByName(modID string, versionName string, ctx *context.Context) *Version {
	cacheKey := "GetModVersionByName_" + modID + "_" + versionName
	if version, ok := dbCache.Get(cacheKey); ok {
		return version.(*Version)
	}

	var version Version
	DBCtx(ctx).First(&version, "mod_id = ? AND version = ?", modID, versionName)

	if version.ID == "" {
		return nil
	}

	dbCache.Set(cacheKey, &version, cache.DefaultExpiration)

	return &version
}

func CreateVersion(version *Version, ctx *context.Context) error {
	var versionCount int64
	DBCtx(ctx).Model(Version{}).Where("mod_id = ? AND version = ?", version.ModID, version.Version).Count(&versionCount)

	if versionCount > 0 {
		return errors.New("this mod already has a version with this name")
	}

	// Allow only new 5 versions per 24h

	var versions []Version
	DBCtx(ctx).Order("created_at asc").Find(&versions, "mod_id = ? AND created_at > ?", version.ModID, time.Now().Add(time.Hour*24*-1))

	if len(versions) >= 5 {
		timeToWait := time.Until(versions[0].CreatedAt.Add(time.Hour * 24)).Minutes()
		return fmt.Errorf("please wait %.0f minutes to post another version", timeToWait)
	}

	version.ID = util.GenerateUniqueID()
	DBCtx(ctx).Create(&version)

	return nil
}

func IncrementVersionDownloads(version *Version, ctx *context.Context) {
	DBCtx(ctx).Model(version).Update("downloads", version.Downloads+1)
}

func GetVersion(versionID string, ctx *context.Context) *Version {
	cacheKey := "GetVersion_" + versionID
	if version, ok := dbCache.Get(cacheKey); ok {
		return version.(*Version)
	}

	var version Version
	DBCtx(ctx).First(&version, "id = ?", versionID)

	if version.ID == "" {
		return nil
	}

	dbCache.Set(cacheKey, &version, cache.DefaultExpiration)

	return &version
}

func GetVersions(limit int, offset int, orderBy string, order string, search string, unapproved bool, ctx *context.Context) []Version {
	var versions []Version
	query := DBCtx(ctx).Limit(limit).Offset(offset).Order(orderBy+" "+order).Where("approved = ? AND denied = ?", !unapproved, false)

	if search != "" {
		query = query.Where("to_tsvector(version) @@ to_tsquery(?)", strings.Replace(search, " ", " & ", -1))
	}

	query.Find(&versions)
	return versions
}

func GetVersionsNew(filter *models.VersionFilter, unapproved bool, ctx *context.Context) []Version {
	hash, err := filter.Hash()
	cacheKey := ""
	if err == nil {
		cacheKey = "GetVersionsNew_" + hash + "_" + fmt.Sprint(unapproved)
		if versions, ok := dbCache.Get(cacheKey); ok {
			return versions.([]Version)
		}
	}

	var versions []Version
	query := DBCtx(ctx).Where("approved = ? AND denied = ?", !unapproved, false)

	if filter != nil {
		query = query.Limit(*filter.Limit).
			Offset(*filter.Offset).
			Order(string(*filter.OrderBy) + " " + string(*filter.Order))

		if filter.Search != nil && *filter.Search != "" {
			query = query.Where("to_tsvector(version) @@ to_tsquery(?)", strings.Replace(*filter.Search, " ", " & ", -1))
		}

		if filter.Fields != nil && len(filter.Fields) > 0 {
			query = query.Select(filter.Fields)
		}
	}

	query.Find(&versions)

	if cacheKey != "" {
		dbCache.Set(cacheKey, versions, cache.DefaultExpiration)
	}

	return versions
}

func GetVersionCount(search string, unapproved bool, ctx *context.Context) int64 {
	var versionCount int64
	query := DBCtx(ctx).Model(Version{}).Where("approved = ? AND denied = ?", !unapproved, false)

	if search != "" {
		query = query.Where("to_tsvector(version) @@ to_tsquery(?)", strings.Replace(search, " ", " & ", -1))
	}

	query.Count(&versionCount)
	return versionCount
}

func GetVersionCountNew(filter *models.VersionFilter, unapproved bool, ctx *context.Context) int64 {
	hash, err := filter.Hash()
	cacheKey := ""
	if err == nil {
		cacheKey = "GetVersionCountNew_" + hash + "_" + fmt.Sprint(unapproved)
		if versionCount, ok := dbCache.Get(cacheKey); ok {
			return versionCount.(int64)
		}
	}

	var versionCount int64
	query := DBCtx(ctx).Model(Version{}).Where("approved = ? AND denied = ?", !unapproved, false)

	if filter != nil {
		if filter.Search != nil && *filter.Search != "" {
			query = query.Where("to_tsvector(version) @@ to_tsquery(?)", strings.Replace(*filter.Search, " ", " & ", -1))
		}
	}

	query.Count(&versionCount)

	if cacheKey != "" {
		dbCache.Set(cacheKey, versionCount, cache.DefaultExpiration)
	}

	return versionCount
}

func GetVersionDependencies(versionID string, ctx *context.Context) []VersionDependency {
	var versionDependencies []VersionDependency
	DBCtx(ctx).Where("version_id = ?", versionID).Find(&versionDependencies)
	return versionDependencies
}
