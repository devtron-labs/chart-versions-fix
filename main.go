package main

import (
	"github.com/devtron-labs/duplicate-chart-versions-fix/sql"
	"log"
)

func main() {
	logger := log.Default()
	cfg, _ := sql.GetConfig()
	dbConnection, _ := sql.NewDbConnection(cfg)
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
	return
}
