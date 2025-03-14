package service

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/1Panel-dev/1Panel/backend/app/dto"
	"github.com/1Panel-dev/1Panel/backend/global"
	"github.com/1Panel-dev/1Panel/backend/utils/cmd"
	fileUtils "github.com/1Panel-dev/1Panel/backend/utils/files"
	"github.com/google/uuid"
)

func (u *SettingService) SystemScan() dto.CleanData {
	var (
		SystemClean dto.CleanData
		treeData    []dto.CleanTree
	)
	fileOp := fileUtils.NewFileOp()

	originalPath := path.Join(global.CONF.System.BaseDir, "1panel_original")
	originalSize, _ := fileOp.GetDirSize(originalPath)
	treeData = append(treeData, dto.CleanTree{
		ID:          uuid.NewString(),
		Label:       "1panel_original",
		Size:        uint64(originalSize),
		IsCheck:     true,
		IsRecommend: true,
		Type:        "1panel_original",
		Children:    loadTreeWithDir(true, "1panel_original", originalPath, fileOp),
	})

	upgradePath := path.Join(global.CONF.System.BaseDir, "1panel/tmp/upgrade")
	upgradeSize, _ := fileOp.GetDirSize(upgradePath)
	treeData = append(treeData, dto.CleanTree{
		ID:          uuid.NewString(),
		Label:       "upgrade",
		Size:        uint64(upgradeSize),
		IsCheck:     false,
		IsRecommend: true,
		Type:        "upgrade",
		Children:    loadTreeWithDir(true, "upgrade", upgradePath, fileOp),
	})

	snapTree := loadSnapshotTree(fileOp)
	snapSize := uint64(0)
	for _, snap := range snapTree {
		snapSize += uint64(snap.Size)
	}
	treeData = append(treeData, dto.CleanTree{
		ID:          uuid.NewString(),
		Label:       "snapshot",
		Size:        snapSize,
		IsCheck:     true,
		IsRecommend: true,
		Type:        "snapshot",
		Children:    snapTree,
	})

	rollBackTree := loadRollBackTree(fileOp)
	rollbackSize := uint64(0)
	for _, rollback := range rollBackTree {
		rollbackSize += uint64(rollback.Size)
	}
	treeData = append(treeData, dto.CleanTree{
		ID:          uuid.NewString(),
		Label:       "rollback",
		Size:        rollbackSize,
		IsCheck:     true,
		IsRecommend: true,
		Type:        "rollback",
		Children:    rollBackTree,
	})

	cachePath := path.Join(global.CONF.System.BaseDir, "1panel/cache")
	cacheSize, _ := fileOp.GetDirSize(cachePath)
	treeData = append(treeData, dto.CleanTree{
		ID:          uuid.NewString(),
		Label:       "cache",
		Size:        uint64(cacheSize),
		IsCheck:     false,
		IsRecommend: false,
		Type:        "cache",
	})

	unusedTree := loadUnusedFile(fileOp)
	unusedSize := uint64(0)
	for _, unused := range unusedTree {
		unusedSize += uint64(unused.Size)
	}
	treeData = append(treeData, dto.CleanTree{
		ID:          uuid.NewString(),
		Label:       "unused",
		Size:        unusedSize,
		IsCheck:     true,
		IsRecommend: true,
		Type:        "unused",
		Children:    unusedTree,
	})
	SystemClean.SystemClean = treeData

	uploadTreeData := loadUploadTree(fileOp)
	SystemClean.UploadClean = append(SystemClean.UploadClean, uploadTreeData...)

	downloadTreeData := loadDownloadTree(fileOp)
	SystemClean.DownloadClean = append(SystemClean.DownloadClean, downloadTreeData...)

	logTree := loadLogTree(fileOp)
	SystemClean.SystemLogClean = append(SystemClean.SystemLogClean, logTree...)

	return SystemClean
}

func (u *SettingService) SystemClean(req []dto.Clean) {
	size := uint64(0)
	restart := false
	for _, item := range req {
		size += item.Size
		switch item.TreeType {
		case "1panel_original":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel_original", item.Name))

		case "upgrade":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/tmp/upgrade", item.Name))

		case "snapshot":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/tmp/system", item.Name))
			dropFileOrDir(path.Join(global.CONF.System.Backup, "system", item.Name))
		case "snapshot_tmp":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/tmp/system", item.Name))
		case "snapshot_local":
			dropFileOrDir(path.Join(global.CONF.System.Backup, "system", item.Name))

		case "rollback":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/tmp/app"))
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/tmp/database"))
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/tmp/website"))
		case "rollback_app":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/tmp/app", item.Name))
		case "rollback_database":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/tmp/database", item.Name))
		case "rollback_website":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/tmp/website", item.Name))

		case "cache":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/cache", item.Name))
			restart = true

		case "unused":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "original"))
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/resource/apps_bak"))
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/tmp/download"))
			files, _ := os.ReadDir(path.Join(global.CONF.System.BaseDir, "1panel/tmp"))
			if len(files) == 0 {
				continue
			}
			for _, file := range files {
				if strings.HasPrefix(file.Name(), "upgrade_") {
					dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/tmp", file.Name()))
				}
			}
		case "old_original":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "original", item.Name))
		case "old_apps_bak":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/resource/apps_bak", item.Name))
		case "old_download":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/tmp/download", item.Name))
		case "old_upgrade":
			if len(item.Name) == 0 {
				files, _ := os.ReadDir(path.Join(global.CONF.System.BaseDir, "1panel/tmp"))
				if len(files) == 0 {
					continue
				}
				for _, file := range files {
					if strings.HasPrefix(file.Name(), "upgrade_") {
						dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/tmp", file.Name()))
					}
				}
			} else {
				dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/tmp", item.Name))
			}

		case "upload":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/uploads", item.Name))
			if len(item.Name) == 0 {
				dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/tmp/upload"))
			}
		case "upload_tmp":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/tmp/upload", item.Name))
		case "upload_app":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/uploads/app", item.Name))
		case "upload_database":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/uploads/database", item.Name))
		case "upload_website":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/uploads/website", item.Name))
		case "upload_directory":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/uploads/directory", item.Name))
		case "download":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/download", item.Name))
		case "download_app":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/download/app", item.Name))
		case "download_database":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/download/database", item.Name))
		case "download_website":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/download/website", item.Name))
		case "download_directory":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/download/directory", item.Name))

		case "system_log":
			if len(item.Name) == 0 {
				files, _ := os.ReadDir(path.Join(global.CONF.System.BaseDir, "1panel/log"))
				if len(files) == 0 {
					continue
				}
				for _, file := range files {
					if file.Name() == "1Panel.log" {
						continue
					}
					dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/log", file.Name()))
				}
			} else {
				dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/log", item.Name))
			}
		case "docker_log":
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/tmp/docker_logs", item.Name))
		case "task_log":
			pathItem := path.Join(global.CONF.System.BaseDir, "1panel/task", item.Name)
			dropFileOrDir(path.Join(global.CONF.System.BaseDir, "1panel/task", item.Name))
			if len(item.Name) == 0 {
				files, _ := os.ReadDir(pathItem)
				if len(files) == 0 {
					continue
				}
				for _, file := range files {
					_ = cronjobRepo.DeleteRecord(cronjobRepo.WithByRecordFile(path.Join(pathItem, file.Name())))
				}
			} else {
				_ = cronjobRepo.DeleteRecord(cronjobRepo.WithByRecordFile(pathItem))
			}
		}
	}

	_ = settingRepo.Update("LastCleanTime", time.Now().Format("2006-01-02 15:04:05"))
	_ = settingRepo.Update("LastCleanSize", fmt.Sprintf("%v", size))
	_ = settingRepo.Update("LastCleanData", fmt.Sprintf("%v", len(req)))

	if restart {
		go func() {
			_, err := cmd.Exec("systemctl restart 1panel.service")
			if err != nil {
				global.LOG.Errorf("restart system port failed, err: %v", err)
			}
		}()
	}
}

func loadSnapshotTree(fileOp fileUtils.FileOp) []dto.CleanTree {
	var treeData []dto.CleanTree
	path1 := path.Join(global.CONF.System.BaseDir, "1panel/tmp/system")
	list1 := loadTreeWithAllFile(true, path1, "snapshot_tmp", path1, fileOp)
	if len(list1) != 0 {
		size, _ := fileOp.GetDirSize(path1)
		treeData = append(treeData, dto.CleanTree{ID: uuid.NewString(), Label: "snapshot_tmp", Size: uint64(size), Children: list1, Type: "snapshot_tmp", IsRecommend: true})
	}

	path2 := path.Join(global.CONF.System.Backup, "system")
	list2 := loadTreeWithAllFile(true, path2, "snapshot_local", path2, fileOp)
	if len(list2) != 0 {
		size, _ := fileOp.GetDirSize(path2)
		treeData = append(treeData, dto.CleanTree{ID: uuid.NewString(), Label: "snapshot_local", Size: uint64(size), Children: list2, Type: "snapshot_local", IsRecommend: true})
	}
	return treeData
}

func loadRollBackTree(fileOp fileUtils.FileOp) []dto.CleanTree {
	var treeData []dto.CleanTree
	path1 := path.Join(global.CONF.System.BaseDir, "1panel/tmp/app")
	list1 := loadTreeWithAllFile(true, path1, "rollback_app", path1, fileOp)
	size1, _ := fileOp.GetDirSize(path1)
	treeData = append(treeData, dto.CleanTree{ID: uuid.NewString(), Label: "rollback_app", Size: uint64(size1), Children: list1, Type: "rollback_app", IsRecommend: true})

	path2 := path.Join(global.CONF.System.BaseDir, "1panel/tmp/website")
	list2 := loadTreeWithAllFile(true, path2, "rollback_website", path2, fileOp)
	size2, _ := fileOp.GetDirSize(path2)
	treeData = append(treeData, dto.CleanTree{ID: uuid.NewString(), Label: "rollback_website", Size: uint64(size2), Children: list2, Type: "rollback_website", IsRecommend: true})

	path3 := path.Join(global.CONF.System.BaseDir, "1panel/tmp/database")
	list3 := loadTreeWithAllFile(true, path3, "rollback_database", path3, fileOp)
	size3, _ := fileOp.GetDirSize(path3)
	treeData = append(treeData, dto.CleanTree{ID: uuid.NewString(), Label: "rollback_database", Size: uint64(size3), Children: list3, Type: "rollback_database", IsRecommend: true})

	return treeData
}

func loadUnusedFile(fileOp fileUtils.FileOp) []dto.CleanTree {
	var treeData []dto.CleanTree
	path1 := path.Join(global.CONF.System.BaseDir, "original")
	list1 := loadTreeWithAllFile(true, path1, "old_original", path1, fileOp)
	if len(list1) != 0 {
		size, _ := fileOp.GetDirSize(path1)
		treeData = append(treeData, dto.CleanTree{ID: uuid.NewString(), Label: "old_original", Size: uint64(size), Children: list1, Type: "old_original"})
	}

	path2 := path.Join(global.CONF.System.BaseDir, "1panel/resource/apps_bak")
	list2 := loadTreeWithAllFile(true, path2, "old_apps_bak", path2, fileOp)
	if len(list2) != 0 {
		size, _ := fileOp.GetDirSize(path2)
		treeData = append(treeData, dto.CleanTree{ID: uuid.NewString(), Label: "old_apps_bak", Size: uint64(size), Children: list2, Type: "old_apps_bak"})
	}

	path3 := path.Join(global.CONF.System.BaseDir, "1panel/tmp/download")
	list3 := loadTreeWithAllFile(true, path3, "old_download", path3, fileOp)
	if len(list3) != 0 {
		size, _ := fileOp.GetDirSize(path3)
		treeData = append(treeData, dto.CleanTree{ID: uuid.NewString(), Label: "old_download", Size: uint64(size), Children: list3, Type: "old_download"})
	}

	path4 := path.Join(global.CONF.System.BaseDir, "1panel/tmp")
	list4 := loadTreeWithDir(true, "old_upgrade", path4, fileOp)
	itemSize := uint64(0)
	for _, item := range list4 {
		itemSize += item.Size
	}
	if len(list4) != 0 {
		treeData = append(treeData, dto.CleanTree{ID: uuid.NewString(), Label: "old_upgrade", Size: itemSize, Children: list4, Type: "old_upgrade"})
	}
	return treeData
}

func loadUploadTree(fileOp fileUtils.FileOp) []dto.CleanTree {
	var treeData []dto.CleanTree

	path0 := path.Join(global.CONF.System.BaseDir, "1panel/tmp/upload")
	list0 := loadTreeWithAllFile(true, path0, "upload_tmp", path0, fileOp)
	size0, _ := fileOp.GetDirSize(path0)
	treeData = append(treeData, dto.CleanTree{ID: uuid.NewString(), Label: "upload_tmp", Size: uint64(size0), Children: list0, Type: "upload_tmp", IsRecommend: true})

	path1 := path.Join(global.CONF.System.BaseDir, "1panel/uploads/app")
	list1 := loadTreeWithAllFile(true, path1, "upload_app", path1, fileOp)
	size1, _ := fileOp.GetDirSize(path1)
	treeData = append(treeData, dto.CleanTree{ID: uuid.NewString(), Label: "upload_app", Size: uint64(size1), Children: list1, Type: "upload_app", IsRecommend: true})

	path2 := path.Join(global.CONF.System.BaseDir, "1panel/uploads/website")
	list2 := loadTreeWithAllFile(true, path2, "upload_website", path2, fileOp)
	size2, _ := fileOp.GetDirSize(path2)
	treeData = append(treeData, dto.CleanTree{ID: uuid.NewString(), Label: "upload_website", Size: uint64(size2), Children: list2, Type: "upload_website", IsRecommend: true})

	path3 := path.Join(global.CONF.System.BaseDir, "1panel/uploads/database")
	list3 := loadTreeWithAllFile(true, path3, "upload_database", path3, fileOp)
	size3, _ := fileOp.GetDirSize(path3)
	treeData = append(treeData, dto.CleanTree{ID: uuid.NewString(), Label: "upload_database", Size: uint64(size3), Children: list3, Type: "upload_database", IsRecommend: true})

	path4 := path.Join(global.CONF.System.BaseDir, "1panel/uploads/directory")
	list4 := loadTreeWithAllFile(true, path4, "upload_directory", path4, fileOp)
	size4, _ := fileOp.GetDirSize(path4)
	treeData = append(treeData, dto.CleanTree{ID: uuid.NewString(), Label: "upload_directory", Size: uint64(size4), Children: list4, Type: "upload_directory", IsRecommend: true})

	path5 := path.Join(global.CONF.System.BaseDir, "1panel/uploads")
	uploadTreeData := loadTreeWithAllFile(true, path5, "upload", path5, fileOp)
	treeData = append(treeData, uploadTreeData...)

	return treeData
}

func loadDownloadTree(fileOp fileUtils.FileOp) []dto.CleanTree {
	var treeData []dto.CleanTree
	path1 := path.Join(global.CONF.System.BaseDir, "1panel/download/app")
	list1 := loadTreeWithAllFile(true, path1, "download_app", path1, fileOp)
	size1, _ := fileOp.GetDirSize(path1)
	treeData = append(treeData, dto.CleanTree{ID: uuid.NewString(), Label: "download_app", Size: uint64(size1), Children: list1, Type: "download_app", IsRecommend: true})

	path2 := path.Join(global.CONF.System.BaseDir, "1panel/download/website")
	list2 := loadTreeWithAllFile(true, path2, "download_website", path2, fileOp)
	size2, _ := fileOp.GetDirSize(path2)
	treeData = append(treeData, dto.CleanTree{ID: uuid.NewString(), Label: "download_website", Size: uint64(size2), Children: list2, Type: "download_website", IsRecommend: true})

	path3 := path.Join(global.CONF.System.BaseDir, "1panel/download/database")
	list3 := loadTreeWithAllFile(true, path3, "download_database", path3, fileOp)
	size3, _ := fileOp.GetDirSize(path3)
	treeData = append(treeData, dto.CleanTree{ID: uuid.NewString(), Label: "download_database", Size: uint64(size3), Children: list3, Type: "download_database", IsRecommend: true})

	path4 := path.Join(global.CONF.System.BaseDir, "1panel/download/directory")
	list4 := loadTreeWithAllFile(true, path4, "download_directory", path4, fileOp)
	size4, _ := fileOp.GetDirSize(path4)
	treeData = append(treeData, dto.CleanTree{ID: uuid.NewString(), Label: "download_directory", Size: uint64(size4), Children: list4, Type: "download_directory", IsRecommend: true})

	path5 := path.Join(global.CONF.System.BaseDir, "1panel/download")
	uploadTreeData := loadTreeWithAllFile(true, path5, "download", path5, fileOp)
	treeData = append(treeData, uploadTreeData...)

	return treeData
}

func loadLogTree(fileOp fileUtils.FileOp) []dto.CleanTree {
	var treeData []dto.CleanTree
	path1 := path.Join(global.CONF.System.BaseDir, "1panel/log")
	list1 := loadTreeWithAllFile(true, path1, "system_log", path1, fileOp)
	size := uint64(0)
	for _, file := range list1 {
		size += file.Size
	}
	treeData = append(treeData, dto.CleanTree{ID: uuid.NewString(), Label: "system_log", Size: uint64(size), Children: list1, Type: "system_log", IsRecommend: true})

	path2 := path.Join(global.CONF.System.BaseDir, "1panel/tmp/docker_logs")
	list2 := loadTreeWithAllFile(true, path2, "docker_log", path2, fileOp)
	size2, _ := fileOp.GetDirSize(path2)
	treeData = append(treeData, dto.CleanTree{ID: uuid.NewString(), Label: "docker_log", Size: uint64(size2), Children: list2, Type: "docker_log", IsRecommend: true})

	path3 := path.Join(global.CONF.System.BaseDir, "1panel/task")
	list3 := loadTreeWithAllFile(false, path3, "task_log", path3, fileOp)
	size3, _ := fileOp.GetDirSize(path3)
	treeData = append(treeData, dto.CleanTree{ID: uuid.NewString(), Label: "task_log", Size: uint64(size3), Children: list3, Type: "task_log"})
	return treeData
}

func loadTreeWithDir(isCheck bool, treeType, pathItem string, fileOp fileUtils.FileOp) []dto.CleanTree {
	var lists []dto.CleanTree
	files, err := os.ReadDir(pathItem)
	if err != nil {
		return lists
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() > files[j].Name()
	})
	for _, file := range files {
		if (treeType == "old_upgrade" || treeType == "upgrade") && !strings.HasPrefix(file.Name(), "upgrade_2023") {
			continue
		}
		if file.IsDir() {
			size, err := fileOp.GetDirSize(path.Join(pathItem, file.Name()))
			if err != nil {
				continue
			}
			item := dto.CleanTree{
				ID:          uuid.NewString(),
				Label:       file.Name(),
				Type:        treeType,
				Size:        uint64(size),
				Name:        strings.TrimPrefix(file.Name(), "/"),
				IsCheck:     isCheck,
				IsRecommend: isCheck,
			}
			if treeType == "upgrade" && len(lists) == 0 {
				item.IsCheck = false
				item.IsRecommend = false
			}
			lists = append(lists, item)
		}
	}
	return lists
}

func loadTreeWithAllFile(isCheck bool, originalPath, treeType, pathItem string, fileOp fileUtils.FileOp) []dto.CleanTree {
	var lists []dto.CleanTree

	files, err := os.ReadDir(pathItem)
	if err != nil {
		return lists
	}
	for _, file := range files {
		if treeType == "system_log" && file.Name() == "1Panel.log" {
			continue
		}
		if (treeType == "upload" || treeType == "download") && file.IsDir() && (file.Name() == "app" || file.Name() == "database" || file.Name() == "website" || file.Name() == "directory") {
			continue
		}
		size := uint64(0)
		name := strings.TrimPrefix(path.Join(pathItem, file.Name()), originalPath+"/")
		if file.IsDir() {
			sizeItem, err := fileOp.GetDirSize(path.Join(pathItem, file.Name()))
			if err != nil {
				continue
			}
			size = uint64(sizeItem)
		} else {
			fileInfo, err := file.Info()
			if err != nil {
				continue
			}
			size = uint64(fileInfo.Size())
		}
		item := dto.CleanTree{
			ID:          uuid.NewString(),
			Label:       file.Name(),
			Type:        treeType,
			Size:        uint64(size),
			Name:        name,
			IsCheck:     isCheck,
			IsRecommend: isCheck,
		}
		if file.IsDir() {
			item.Children = loadTreeWithAllFile(isCheck, originalPath, treeType, path.Join(pathItem, file.Name()), fileOp)
		}
		lists = append(lists, item)
	}
	return lists
}

func dropFileOrDir(itemPath string) {
	global.LOG.Debugf("drop file %s", itemPath)
	if err := os.RemoveAll(itemPath); err != nil {
		global.LOG.Errorf("drop file %s failed, err %v", itemPath, err)
	}
}
