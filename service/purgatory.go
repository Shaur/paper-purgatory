package service

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"paper/purgatory/dto"
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

	UploadTempFile(source *multipart.FileHeader) (*os.File, string, error)

	SaveMeta(meta dto.NewMeta) *model.PurgatoryItem
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

func (s *purgatoryService) UploadTempFile(source *multipart.FileHeader) (*os.File, string, error) {
	src, err := source.Open()
	if err != nil {
		return nil, "", err
	}

	filename := source.Filename
	destPath := filepath.Join(s.filesPath, filename)
	dest, err := os.Create(destPath)
	if err != nil {
		return nil, "", err
	}

	_, err = io.Copy(dest, src)
	if err != nil {
		return nil, "", err
	}

	return dest, destPath, nil
}

func (s *purgatoryService) SaveMeta(meta dto.NewMeta) *model.PurgatoryItem {
	archiveMeta := &model.ArchiveMeta{
		SeriesName: meta.Title,
		Number:     meta.Number,
		PagesCount: 0,
	}

	item := model.PurgatoryItem{Meta: archiveMeta}
	s.database.Create(&item)

	return &item
}
