package db

import (
	"einoproject/internal/config"
	"einoproject/internal/entity"
	"errors"
	"fmt"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func NewDB(dbcfg config.DbConfig) (*gorm.DB, error) {
	if strings.TrimSpace(dbcfg.DBName) == "" {
		return nil, errors.New("database name is required")
	}

	rootDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
		dbcfg.User, dbcfg.Password, dbcfg.Host, dbcfg.Port)

	rootDB, err := gorm.Open(mysql.Open(rootDSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	createDBSQL := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		escapeIdentifier(dbcfg.DBName),
	)
	if err := rootDB.Exec(createDBSQL).Error; err != nil {
		return nil, err
	}
	if err := CloseDB(rootDB); err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbcfg.User, dbcfg.Password, dbcfg.Host, dbcfg.Port, dbcfg.DBName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func escapeIdentifier(identifier string) string {
	return strings.ReplaceAll(identifier, "`", "``")
}

func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&entity.Account{},
		&entity.Conversation{},
		&entity.Message{},
	); err != nil {
		return err
	}
	return nil
}
func CloseDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
