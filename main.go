package main

import (
	"fmt"
	"github.com/devtron-labs/duplicate-chart-versions-fix/sql"
	"github.com/go-pg/pg"
	"log"
)

func main() {
	logger := log.Default()
	cfg, _ := sql.GetConfig()
	dbConnection, _ := sql.NewDbConnection(cfg)
	cleanDuplicateAppStoreApplicationVersions(dbConnection, logger)
	cleanDuplicateAppStoreCharts(dbConnection, logger)
	return
}

func cleanDuplicateAppStoreApplicationVersions(dbConnection *pg.DB, logger *log.Logger) {
	appStoreApplicationVersionImpl := sql.NewAppStoreApplicationVersionRepositoryImpl(dbConnection)
	appStoreApplicationVersions, err := appStoreApplicationVersionImpl.FindAll()
	if err != nil {
		logger.Println("error in fetching app store application version", "err", err)
		return
	}
	uniqueAppVersionId := make(map[string]map[string]int)
	for _, appStoreApplicationVersion := range appStoreApplicationVersions {
		if _, ok := uniqueAppVersionId[appStoreApplicationVersion.Name]; !ok {
			uniqueAppVersionId[appStoreApplicationVersion.Name] = make(map[string]int)
		}
		if _, appVersionOk := uniqueAppVersionId[appStoreApplicationVersion.Name][appStoreApplicationVersion.Version]; !appVersionOk {
			uniqueAppVersionId[appStoreApplicationVersion.Name][appStoreApplicationVersion.Version] = appStoreApplicationVersion.Id
		}
	}
	for _, appStoreApplicationVersion := range appStoreApplicationVersions {
		uniqueAppStoreVersionId := uniqueAppVersionId[appStoreApplicationVersion.Name][appStoreApplicationVersion.Version]
		if appStoreApplicationVersion.Id != uniqueAppStoreVersionId {
			logger.Println("Found duplicate app store version", "app store version id", appStoreApplicationVersion.Id)
			installedAppVersions, err := appStoreApplicationVersionImpl.GetInstalledAppVersionByAppStoreApplicationVersionId(appStoreApplicationVersion.Id)
			if err != nil {
				logger.Println("error in fetching installed app versions for appStoreApplicationVersionId", "appStoreApplicationVersionId", appStoreApplicationVersion.Id, "err", err)
				return
			}
			for _, installedAppVersion := range installedAppVersions {
				logger.Println("found app in duplicate version, updating installed app version", "installedAppVersionId", installedAppVersion.Id)
				installedAppVersion.AppStoreApplicationVersionId = uniqueAppStoreVersionId
				installedAppVersion, err = appStoreApplicationVersionImpl.UpdateInstalledAppVersion(installedAppVersion, nil)
				if err != nil {
					logger.Println("error in updating installed app version", "err", err, "installedAppVersionId", installedAppVersion.Id)
					return
				}
			}
			if appStoreApplicationVersion.Latest == true {
				err = appStoreApplicationVersionImpl.UpdateAppStoreApplicationVersion(uniqueAppStoreVersionId)
				if err != nil {
					logger.Println("error in updating application version to latest ", "err", err)
					return
				}
			}
			logger.Println("deleting app store application version by appStoreApplicationVersionId", "appStoreApplicationVersionId", appStoreApplicationVersion.Id)
			err = appStoreApplicationVersionImpl.DeleteAppStoreApplicationVersion(appStoreApplicationVersion.Id)
			if err != nil {
				logger.Println("error in deleting app store application version", "err", err)
				return
			}
		}
	}
	logger.Println("app store application version clean up successful")
}

func cleanDuplicateAppStoreCharts(dbConnection *pg.DB, logger *log.Logger) {
	appStoreRepository := sql.NewAppStoreApplicationVersionRepositoryImpl(dbConnection)
	appStoreAllApps, err := appStoreRepository.FindAllAppStores()
	if err != nil {
		logger.Println("error in fetching from app_store", "err", err)
		return
	}
	activeAppStoreMap := make(map[string]*sql.AppStore)
	AppVersionCount := make(map[string]int)
	for _, appStore := range appStoreAllApps {
		appStoreApplicationVersions, err := appStoreRepository.FindChartVersionByAppStoreId(appStore.Id)
		if err != nil {
			logger.Println("error in fetching app store app version by app store id", "err", err, "app_store_id", appStore.Id)
			continue
		}
		var uniqueKey string
		if appStore.ChartRepoId == 0 {
			uniqueKey = fmt.Sprintf("%s-%s", appStore.Name, appStore.DockerArtifactStoreId)
		} else {
			uniqueKey = fmt.Sprintf("%s-%v", appStore.Name, appStore.ChartRepoId)
		}
		if _, ok := AppVersionCount[uniqueKey]; !ok {
			AppVersionCount[uniqueKey] = len(appStoreApplicationVersions)
			activeAppStoreMap[uniqueKey] = appStore
		} else if len(appStoreApplicationVersions) > AppVersionCount[uniqueKey] {
			AppVersionCount[uniqueKey] = len(appStoreApplicationVersions)
			activeAppStoreMap[uniqueKey] = appStore
		}
	}
	logger.Print("updating charts to correct repo if they are in repo which will be deleted")
	for _, appStore := range appStoreAllApps {
		var uniqueKey string
		if appStore.ChartRepoId == 0 {
			uniqueKey = fmt.Sprintf("%s-%s", appStore.Name, appStore.DockerArtifactStoreId)
		} else {
			uniqueKey = fmt.Sprintf("%s-%v", appStore.Name, appStore.ChartRepoId)
		}
		if appStore.Id != activeAppStoreMap[uniqueKey].Id {
			installedAppVersions, err := appStoreRepository.GetInstalledAppVersionByAppStoreId(appStore.Id)
			if err != nil {
				logger.Println("error in fetching installed app versions by appStoreId", "err", err, "appStoreId", appStore.Id)
				return
			}
			for _, installedAppVersion := range installedAppVersions {
				activeAppStoreApplicationVersion, err := appStoreRepository.FindAppStoreVersionByAppStoreIdAndChartVersion(activeAppStoreMap[uniqueKey].Id, installedAppVersion.AppStoreApplicationVersion.Name, installedAppVersion.AppStoreApplicationVersion.Version)
				if err != nil {
					logger.Println("error in fetching active app store application version by id", "err", err)
					continue
				}
				if activeAppStoreApplicationVersion == nil {
					continue
				}
				logger.Println(" migrating from old applicationVersionId  to new ", "oldApplicationVersionId", installedAppVersion.AppStoreApplicationVersionId, "newApplicationVersionId", activeAppStoreApplicationVersion.Id)
				installedAppVersion.AppStoreApplicationVersionId = activeAppStoreApplicationVersion.Id
				installedAppVersion, err = appStoreRepository.UpdateInstalledAppVersion(installedAppVersion, nil)
				if err != nil {
					logger.Println("error in updating installed app version", "err", err, "installedAppVersionId", installedAppVersion.Id)
					return
				}
			}
		}
	}
	logger.Println("deleting duplicate charts")
	var appsToBeDeleted []*sql.AppStore
	for _, appStore := range appStoreAllApps {
		var uniqueKey string
		if appStore.ChartRepoId == 0 {
			uniqueKey = fmt.Sprintf("%s-%s", appStore.Name, appStore.DockerArtifactStoreId)
		} else {
			uniqueKey = fmt.Sprintf("%s-%v", appStore.Name, appStore.ChartRepoId)
		}
		if appStore.Id != activeAppStoreMap[uniqueKey].Id {
			appsToBeDeleted = append(appsToBeDeleted, appStore)
		}
	}
	err = appStoreRepository.Delete(appsToBeDeleted)
	if err != nil {
		logger.Println("error in marking app store version as deleted", "err", err)
		return
	}
	logger.Println("duplicate app stores deleted")
}
