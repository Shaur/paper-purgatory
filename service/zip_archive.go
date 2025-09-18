package service

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"paper/purgatory/model"
	"paper/purgatory/utils"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type CbzTool struct {
	baseArchiveTool
}

func NewCbzTool(fileName string) *CbzTool {
	return &CbzTool{
		baseArchiveTool: baseArchiveTool{fileName: fileName},
	}
}

func (c *CbzTool) GetMeta(file *os.File, size int64) (*model.ArchiveMeta, error) {
	zipReader, err := zip.NewReader(file, size)
	if err != nil {
		return nil, fmt.Errorf("failed to create zip reader: %v", err)
	}

	xmlFile, err := os.CreateTemp("", "ComicInfo_*.xml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %v", err)
	}

	defer utils.HandleRemove(os.Remove, xmlFile.Name())
	defer utils.HandleClose(xmlFile.Close)

	var descriptors []string
	var xmlFound bool

	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		descriptors = append(descriptors, file.Name)

		lowerCaseFileName := strings.ToLower(file.Name)
		if !xmlFound && strings.Contains(lowerCaseFileName, infoFileName) {
			xmlFound = true
			srcFile, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open XML file: %v", err)
			}
			defer utils.HandleClose(srcFile.Close)

			if _, err := io.Copy(xmlFile, srcFile); err != nil {
				xmlFound = false
			}
		}
	}

	if len(descriptors) == 0 {
		return nil, fmt.Errorf("empty archive %v", c.fileName)
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

func (c *CbzTool) Extract(file *os.File, destination string) error {
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}

	size := info.Size()
	zipReader, err := zip.NewReader(file, size)
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %v", err)
	}

	if err := os.MkdirAll(destination, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %v", err)
	}

	var imageFiles []string
	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() || strings.HasSuffix(strings.ToLower(file.Name), ".xml") {
			continue
		}

		srcFile, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open file %s: %v", file.Name, err)
		}
		defer utils.HandleClose(srcFile.Close)

		// Use original filename temporarily
		tempFilename := filepath.Base(file.Name)
		tempPath := filepath.Join(destination, tempFilename)

		dstFile, err := os.Create(tempPath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %v", tempPath, err)
		}

		defer utils.HandleClose(dstFile.Close)

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return fmt.Errorf("failed to copy file %s: %v", file.Name, err)
		}

		imageFiles = append(imageFiles, tempFilename)
	}

	// Sort files by name length then alphabetically
	sort.Slice(imageFiles, func(i, j int) bool {
		if len(imageFiles[i]) != len(imageFiles[j]) {
			return len(imageFiles[i]) < len(imageFiles[j])
		}
		return imageFiles[i] < imageFiles[j]
	})

	// Calculate digits needed for padding
	digits := len(strconv.Itoa(len(imageFiles)))

	// Second pass: rename files to sequential names
	for index, oldFilename := range imageFiles {
		oldPath := filepath.Join(destination, oldFilename)
		newFilename := fmt.Sprintf("%0*d.jpg", digits, index)
		newPath := filepath.Join(destination, newFilename)

		// Read the old file
		content, err := os.ReadFile(oldPath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %v", oldPath, err)
		}

		// Write to new file
		if err := os.WriteFile(newPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %v", newPath, err)
		}

		// Remove old file
		if err := os.Remove(oldPath); err != nil {
			return fmt.Errorf("failed to remove old file %s: %v", oldPath, err)
		}
	}

	return nil
}
