package model

type PurgatoryItem struct {
	ID   int64       `gorm:"unique;primaryKey;autoIncrement" json:"id"`
	Meta ArchiveMeta `gorm:"type:jsonb;serializer:json" json:"meta"`
}

func (PurgatoryItem) TableName() string {
	return "purgatory"
}

type ArchiveMeta struct {
	SeriesName string `json:"seriesName"`
	Number     string `json:"number"`
	Summary    string `json:"summary"`
	Publisher  string `json:"publisher"`
	PagesCount int32  `json:"pagesCount"`
}

type User struct {
	Username string
}

func (User) TableName() string {
	return "user_data"
}
