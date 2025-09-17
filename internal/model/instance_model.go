package model

type Instance struct {
	ID       int64  `gorm:"column:id" json:"-"`
	UUID     string `gorm:"column:uuid" json:"uuid"`
	Version  string `gorm:"column:version" json:"version"`
	LastSeen int64  `gorm:"column:last_seen" json:"last_seen"`
}
