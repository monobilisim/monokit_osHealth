// go:build osHealth

package vlib

import (
	lib "github.com/monobilisim/monokit2/lib"
)

func UpsertVersion(name string, version string, versionMulti string) {
	var Version lib.Version

	Version.Name = name
	Version.Version = version
	Version.VersionMulti = versionMulti

	var existing []lib.Version

	err := lib.DB.Model(&lib.Version{}).Where("name = ?", name).Find(&existing).Error
	if err != nil {
		return
	}

	if len(existing) <= 0 {
		lib.DB.Create(&Version)
		return
	}

	if len(existing) > 0 {
		lib.DB.Model(&lib.Version{}).Where("name = ?", name).Updates(&Version)
		return
	}
}
