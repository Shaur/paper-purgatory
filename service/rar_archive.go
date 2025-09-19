package service

import (
	"fmt"
	"io"
	"os"
	"paper/purgatory/model"
	"paper/purgatory/utils"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/nwaples/rardecode/v2"
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

	rarReader, err := rardecode.NewReader(file)
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
	var descriptors []string

	for {
		header, err := rarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read RAR entry: %v", err)
		}

		descriptors = append(descriptors, header.Name)

		lowerCaseFileName := strings.ToLower(header.Name)
		if strings.Contains(lowerCaseFileName, infoFileName) {
			xmlFound = true
			if _, err := io.Copy(xmlFile, rarReader); err != nil {
				return nil, fmt.Errorf("failed to extract XML content: %v", err)
			}
		} else {
			io.Copy(io.Discard, rarReader)
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

	rarReader, err := rardecode.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create RAR reader: %v", err)
	}

	// Create destination directory
	if err := os.MkdirAll(destination, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %v", err)
	}

	// Extract files
	var extractedFiles []string

	for {
		header, err := rarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read RAR entry: %v", err)
		}

		// Skip XML files
		if strings.HasSuffix(strings.ToLower(header.Name), ".xml") {
			io.Copy(io.Discard, rarReader)
			continue
		}

		// Create output file
		outputPath := filepath.Join(destination, filepath.Base(header.Name))
		outputFile, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file %s: %v", outputPath, err)
		}

		// Extract file content
		if _, err := io.Copy(outputFile, rarReader); err != nil {
			return fmt.Errorf("failed to extract file %s: %v", header.Name, err)
		}

		utils.HandleClose(outputFile.Close)

		extractedFiles = append(extractedFiles, outputPath)
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
		oldPath := filepath.Join(oldFile)

		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("failed to rename file %s to %s: %v", oldPath, newPath, err)
		}
	}

	return nil
}
