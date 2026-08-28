package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func ConnectDB(dsn string) (*gorm.DB, error) {
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	if err := gormDB.AutoMigrate(&Order{}, &OrderItem{}, &Cart{}, &CartItem{}); err != nil {
		return nil, err
	}

	return gormDB, nil
}
