package service

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html/charset"

	"paper/purgatory/model"
)

type ArchiveTool interface {
	GetMeta(input *os.File, size int64) (*model.ArchiveMeta, error)
	Extract(input *os.File, destination string) error
}

type baseArchiveTool struct {
	fileName string
}

var (
	seriesNameRegex = regexp.MustCompile(`^[A-Za-z\d)(.\-" ']+ \d{1,3}`)
	numberRegex     = regexp.MustCompile(`\d+`)

	infoFileName = "comicinfo.xml"
)

func (b *baseArchiveTool) extractSeriesNameFromFileName(fileName string) string {
	normalized := strings.ReplaceAll(fileName, "_", " ")
	match := seriesNameRegex.FindString(normalized)
	if match == "" {
		return fileName
	}

	lastIndexOfSpace := strings.LastIndex(match, " ")
	if lastIndexOfSpace == -1 {
		return fileName
	}

	return strings.TrimSpace(match[:lastIndexOfSpace])
}

func (b *baseArchiveTool) extractMetaFromXml(path string) (*model.ArchiveMeta, error) {
	xmlContent, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read XML file: %v", err)
	}

	type ComicInfo struct {
		XMLName   xml.Name `xml:"ComicInfo"`
		Title     string   `xml:"Title"`
		Number    string   `xml:"Number"`
		Issue     string   `xml:"Issue"`
		Series    string   `xml:"Series"`
		Publisher string   `xml:"Publisher"`
		Summary   string   `xml:"Summary"`
	}

	// Handle XML encoding (common issue with ComicInfo files)
	reader := bytes.NewReader(xmlContent)
	decoder := xml.NewDecoder(reader)
	decoder.CharsetReader = charset.NewReaderLabel

	var comicInfo ComicInfo
	if err := decoder.Decode(&comicInfo); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %v", err)
	}

	title := strings.TrimSpace(comicInfo.Series)
	if title == "" {
		title = strings.TrimSpace(comicInfo.Title)
	}

	number := ""
	for _, n := range []string{
		strings.TrimSpace(comicInfo.Number),
		strings.TrimSpace(comicInfo.Issue),
		strings.TrimSpace(comicInfo.Title),
	} {
		if n != "" {
			number = n
			break
		}
	}

	return &model.ArchiveMeta{
		SeriesName: title,
		Publisher:  strings.TrimSpace(comicInfo.Publisher),
		Summary:    strings.TrimSpace(comicInfo.Summary),
		Number:     b.extractFirstNumber(number),
		PagesCount: 0,
	}, nil
}

func (b *baseArchiveTool) extractNumberFromFileName(fileName string) string {
	normalized := strings.ReplaceAll(fileName, "_", " ")
	match := seriesNameRegex.FindString(normalized)
	if match == "" {
		return fileName
	}

	lastIndexOfSpace := strings.LastIndex(match, " ")
	if lastIndexOfSpace == -1 {
		return fileName
	}

	numberStr := strings.TrimSpace(match[lastIndexOfSpace:])
	return b.parseNumber(numberStr)
}

func (b *baseArchiveTool) extractNumber(fileName, seriesName string) string {
	if len(fileName) <= len(seriesName)+1 {
		return ""
	}

	substring := fileName[len(seriesName)+1:]
	indexOfSpace := strings.Index(substring, " ")
	if indexOfSpace == -1 {
		return ""
	}

	numberStr := strings.TrimSpace(substring[:indexOfSpace])
	match := numberRegex.FindString(numberStr)
	if match == "" {
		return ""
	}

	return b.parseNumber(match)
}

func (b *baseArchiveTool) extractFirstNumber(str string) string {
	match := numberRegex.FindString(str)
	if match == "" {
		return ""
	}
	return b.parseNumber(match)
}

func (b *baseArchiveTool) parseNumber(numberStr string) string {
	// Try parsing as integer first
	if num, err := strconv.Atoi(numberStr); err == nil {
		return strconv.Itoa(num)
	}

	// Try parsing as float
	if num, err := strconv.ParseFloat(numberStr, 64); err == nil {
		return fmt.Sprintf("%g", num)
	}

	// Return as string if parsing fails
	return numberStr
}

func (b *baseArchiveTool) crossNames(name1, name2 string) string {
	minLen := min(len(name1), len(name2))
	name1Lower := strings.ToLower(name1)
	name2Lower := strings.ToLower(name2)

	var result strings.Builder
	for i := 0; i < minLen; i++ {
		if name1Lower[i] == ' ' || name2Lower[i] == ' ' {
			result.WriteRune(' ')
			continue
		}

		if name1Lower[i] != name2Lower[i] {
			break
		}

		result.WriteByte(name1[i])
	}

	return strings.TrimSpace(result.String())
}

func (b *baseArchiveTool) resolveNumber(seriesName string) string {
	number1 := b.extractNumber(b.fileName, seriesName)
	number2 := b.extractNumberFromFileName(b.fileName)

	if number1 != "" {
		return number1
	} else if number2 != "" {
		return number2
	}

	return ""
}

func (b *baseArchiveTool) resolveSeriesName(firstPage string) string {
	seriesName1 := b.crossNames(b.fileName, firstPage)
	seriesName2 := b.extractSeriesNameFromFileName(b.fileName)

	seriesName := b.fileName
	if seriesName1 != b.fileName && seriesName1 != "" {
		seriesName = seriesName1
	}
	if seriesName2 != b.fileName && seriesName2 != "" && len(seriesName2) < len(seriesName) {
		seriesName = seriesName2
	}

	return seriesName
}
