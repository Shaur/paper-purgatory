package dto

import "github.com/golang-sql/civil"

type ApproveRequest struct {
	SeriesUpdate SeriesUpdateRequest `json:"seriesUpdate"`
	IssueUpdate  IssueUpdateRequest  `json:"issueUpdate"`
}

type SeriesUpdateRequest struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Publisher string `json:"publisher"`
}

type IssueUpdateRequest struct {
	Number          string     `json:"number"`
	Summary         string     `json:"summary"`
	PublicationDate civil.Date `json:"publicationDate"`
	PagesCount      int32      `json:"pagesCount"`
}
