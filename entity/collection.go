package entity

import (
	"time"

	"github.com/google/uuid"
)

type Collection struct {
	ID          uuid.UUID
	Name        string
	Description string
	Owner       string
	Created     time.Time
}

func NewCollection(name, description, owner string) *Collection {
	return &Collection{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		Owner:       owner,
		Created:     time.Now(),
	}
}
