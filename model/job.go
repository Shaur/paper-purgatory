package model

import "time"

type Job struct {
	Id           int64 `gorm:"primaryKey"`
	FileName     string
	FilePath     string
	Status       string
	UpdateTime   time.Time
	UploadedBy   string
	ErrorMessage *string
	Attempts     int
}

func (Job) TableName() string { return "job" }
