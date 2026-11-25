package service

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"paper/purgatory/dto"
	"paper/purgatory/model"
	"paper/purgatory/utils"
	"path/filepath"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

var ErrUnsupportedFileType = errors.New("unsupported file type")

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

	existsItem := model.PurgatoryItem{}
	result := s.database.Where("meta ->> 'seriesName' like ? and meta ->> 'number' = ?", "%"+meta.SeriesName+"%", meta.Number).First(&existsItem)
	item := model.PurgatoryItem{Meta: meta}
	if result.Error != nil {
		s.database.Create(&item)
	} else {
		item.ID = existsItem.ID
		s.database.Save(&item)
	}

	err = tool.Extract(input, filepath.Join(s.filesPath, strconv.FormatInt(item.ID, 10)))
	if err != nil {
		return err
	}

	return nil
}

func _checkExtension(src multipart.File) error {
	header := make([]byte, 4)
	if _, err := io.ReadFull(src, header); err != nil {
		return err
	}

	isZip := bytes.HasSuffix(header, []byte("PK\x03\x04"))   //CBZ
	isRar := bytes.HasSuffix(header, []byte("Rar!\x1A\x07")) //CBR (RAR 4\5)

	if !isZip && !isRar {
		return ErrUnsupportedFileType
	}

	if _, err := src.Seek(0, 0); err != nil {
		return err
	}

	return nil
}

func (s *purgatoryService) UploadTempFile(source *multipart.FileHeader) (*os.File, string, error) {
	src, err := source.Open()
	if err != nil {
		return nil, "", err
	}

	if err = _checkExtension(src); err != nil {
		return nil, "", err
	}

	defer utils.HandleClose(src.Close)

	filename := filepath.Base(source.Filename)
	destPath := filepath.Join(s.filesPath, filename)

	if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(s.filesPath)+string(os.PathSeparator)) {
		return nil, "", fmt.Errorf("path escapes files directory: %s", destPath)
	}

	dest, err := os.Create(destPath)
	if err != nil {
		return nil, "", err
	}

	_, err = io.Copy(dest, src)
	if err != nil {
		_ = dest.Close()
		_ = os.Remove(destPath)
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
