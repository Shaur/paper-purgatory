package service

import (
	"fmt"
	"os"
	"paper/purgatory/model"
	"path/filepath"
	"strconv"

	"gorm.io/gorm"
)

type purgatoryService struct {
	database  *gorm.DB
	filesPath string
}

type PurgatoryService interface {
	GetAll() *[]model.PurgatoryItem

	Save(input *os.File, name string) error
}

func Init(database *gorm.DB, filesPath string) PurgatoryService {
	return &purgatoryService{database: database, filesPath: filesPath}
}

func (s *purgatoryService) GetAll() *[]model.PurgatoryItem {
	var items []model.PurgatoryItem
	s.database.Find(&items)

	return &items
}

func (s *purgatoryService) Save(input *os.File, name string) error {
	ext := filepath.Ext(input.Name())
	var tool ArchiveTool
	if ext == ".cbr" {
		tool = NewCbrTool(name)
	} else if ext == ".cbz" {
		tool = NewCbzTool(name)
	} else {
		return fmt.Errorf("unsupported format")
	}

	fileStat, err := input.Stat()
	if err != nil {
		return err
	}

	meta, err := tool.GetMeta(input, fileStat.Size())
	if err != nil {
		return err
	}

	item := model.PurgatoryItem{Meta: meta}
	s.database.Create(&item)

	err = tool.Extract(input, filepath.Join(s.filesPath, strconv.FormatInt(item.ID, 10)))
	if err != nil {
		return err
	}

	return nil
}
