package database

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(dns string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dns))
	if err != nil {
		panic(err)
	}
	err = db.AutoMigrate(&Profile{})
	if err != nil {
		log.Fatal(err)
	}
	return db
}
