package models

import "time"

type UserNotification struct {
	ID           uint       `json:"id"`
	UserID       uint       `json:"user_id"`
	ActorUserID  *uint      `json:"-" db:"actor_user_id" gorm:"column:actor_user_id"`
	Actor        *User      `json:"-" gorm:"foreignKey:ActorUserID"`
	EventType    string     `json:"event_type"`
	Title        string     `json:"title"`
	Message      string     `json:"message"`
	ResourceType string     `json:"resource_type,omitempty"`
	ResourceID   *uint      `json:"resource_id,omitempty"`
	ReadAt       *time.Time `json:"read_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (UserNotification) TableName() string { return "user_notifications" }
