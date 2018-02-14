package main

import "github.com/jinzhu/gorm"

type Transaction struct {
	gorm.Model
	Txid string
	Hash string
	Version uint
	Size uint
	Vsize uint
	Locktime uint
	Hex string
	Blockhash string
	Time uint
	Blocktime uint
	Addresses []Address `gorm:"many2many:address_transaction;"`
}
