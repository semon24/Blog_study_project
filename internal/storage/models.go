package storage

import "time"

type Author struct {
	ID       int64 `gorm:"primaryKey;autoIncrement:false"`
	Nickname string
	PhotoURL string
}

type Post struct {
	ID        int64 `gorm:"primaryKey"`
	AuthorID  int64 `gorm:"not null;index"`
	Author    Author
	Body      string `gorm:"type:text;not null"`
	CreatedAt time.Time
}