package model

import "gorm.io/gorm"

type Greeter struct {
	gorm.Model
	Name string `gorm:"type:varchar(255);not null;comment:名称"`
}
