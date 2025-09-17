package model

type RateLimit struct {
	ID     int64  `gorm:"column:id"`
	IP     string `gorm:"column:ip"`
	Count  int64  `gorm:"column:count"`
	Expire int64  `gorm:"column:expire"`
}
