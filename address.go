package main

import "github.com/jinzhu/gorm"

type Address struct {
	gorm.Model
	Hash string
	Balance float64
	Transactions []Transaction `gorm:"many2many:address_transaction;"`
}
