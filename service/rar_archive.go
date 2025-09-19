package service

import (
	"fmt"
	"os"
	"paper/purgatory/model"
	"paper/purgatory/utils"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gen2brain/go-unarr"
)

type CbrTool struct {
	baseArchiveTool
}

func NewCbrTool(fileName string) *CbrTool {
	return &CbrTool{
		baseArchiveTool: baseArchiveTool{fileName: fileName},
	}
}

func (c *CbrTool) GetMeta(file *os.File, _ int64) (*model.ArchiveMeta, error) {
	// Reset file pointer to beginning
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek file: %v", err)
	}

	rarReader, err := unarr.NewArchiveFromReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create RAR reader: %v", err)
	}

	// Create temp file for XML
	xmlFile, err := os.CreateTemp("", "ComicInfo_*.xml")
	if err != nil {
		return nil, fmt.Errorf("failed to create XML temp file: %v", err)
	}

	defer utils.HandleRemove(os.Remove, xmlFile.Name())
	defer utils.HandleClose(xmlFile.Close)

	var xmlFound bool

	descriptors, err := rarReader.List()
	if err != nil {
		return nil, err
	}

	for _, header := range descriptors {
		lowerCaseFileName := strings.ToLower(header)
		if strings.Contains(lowerCaseFileName, infoFileName) {
			xmlFound = true
			err = rarReader.EntryFor(header)
			if err != nil {
				return nil, fmt.Errorf("failed to extract XML content: %v", err)
			}

			data, err := rarReader.ReadAll()
			if err != nil {
				return nil, fmt.Errorf("failed to extract XML content: %v", err)
			}

			if _, err := xmlFile.Write(data); err != nil {
				return nil, fmt.Errorf("failed to extract XML content: %v", err)
			}

		}
	}

	if xmlFound {
		meta, err := c.extractMetaFromXml(xmlFile.Name())
		if err != nil {
			return nil, err
		}

		meta.PagesCount = len(descriptors) - 1
		return meta, nil
	}

	seriesName := c.resolveSeriesName(descriptors[0])

	return &model.ArchiveMeta{
		SeriesName: seriesName,
		Number:     c.resolveNumber(seriesName),
		PagesCount: len(descriptors),
	}, nil
}

func (c *CbrTool) Extract(file *os.File, destination string) error {
	// Reset file pointer to beginning
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek file: %v", err)
	}

	rarReader, err := unarr.NewArchiveFromReader(file)
	if err != nil {
		return fmt.Errorf("failed to create RAR reader: %v", err)
	}

	// Create destination directory
	if err := os.MkdirAll(destination, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %v", err)
	}

	// Extract files
	extractedFiles, err := rarReader.Extract(destination)
	if err != nil {
		return fmt.Errorf("failed to extract archive")
	}

	// Rename files to sequential order
	if len(extractedFiles) > 0 {
		return renameFiles(destination, extractedFiles)
	}

	return nil
}

func renameFiles(destination string, extractedFiles []string) error {
	sort.Strings(extractedFiles)

	digits := len(strconv.Itoa(len(extractedFiles)))

	for index, oldFile := range extractedFiles {
		ext := filepath.Ext(oldFile)
		newFilename := fmt.Sprintf("%0*d%s", digits, index, ext)
		newPath := filepath.Join(destination, newFilename)
		oldPath := filepath.Join(destination, oldFile)

		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("failed to rename file %s to %s: %v", oldPath, newPath, err)
		}
	}

	return nil
}
