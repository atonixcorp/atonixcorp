package database

import (
	"atonixcorp/api/internal/models"
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() *gorm.DB {
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	port := os.Getenv("DB_PORT")

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		host, user, password, dbname, port,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf(" Failed to connect to PostgreSQL: %v", err)
	}

	log.Println(" Connected to PostgreSQL")

	migrate(db)

	DB = db
	return db
}

func migrate(db *gorm.DB) {
	err := db.AutoMigrate(
		&models.User{},
		&models.ContactMessage{},
		&models.BlogPost{},
	)

	if err != nil {
		log.Fatalf(" Migration failed: %v", err)
	}

	log.Println("Database migrations completed")
}
