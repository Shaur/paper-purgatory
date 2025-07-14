package service

import (
	"gorm.io/gorm"
	"paper/purgatory/model"
)

type purgatoryService struct {
	database *gorm.DB
}

type PurgatoryService interface {
	GetAll() *[]model.PurgatoryItem
}

func Init(database *gorm.DB) PurgatoryService {
	return &purgatoryService{database: database}
}

func (s *purgatoryService) GetAll() *[]model.PurgatoryItem {
	var items []model.PurgatoryItem
	s.database.Find(&items)

	return &items
}
