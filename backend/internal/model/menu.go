package model

import "time"

// Menu = satu baris di tabel menus
type Menu struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string    `json:"name"`
	ParentID  *string   `gorm:"type:uuid" json:"parent_id"`
	Depth     int       `json:"depth"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MenuNode = Menu + anak-anaknya, dipakai untuk response berbentuk tree
type MenuNode struct {
	Menu
	Children []*MenuNode `json:"children"`
}