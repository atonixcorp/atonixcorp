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
		&models.Tenant{},
		&models.TenantMembership{},
		&models.Project{},
		&models.ProjectMembership{},
		&models.ContainerDeployment{},
		&models.ContainerDeploymentRevision{},
		&models.ContainerRegistry{},
		&models.ContainerDeploymentEvent{},
		&models.StorageVolume{},
		&models.StorageQuota{},
		&models.StorageOperation{},
		&models.StorageAuditEvent{},
		&models.SecurityPolicy{},
		&models.SecurityAuditEvent{},
		&models.ObservabilityResource{},
		&models.ContactMessage{},
		&models.SupportTicket{},
		&models.BlogPost{},
	)

	if err != nil {
		log.Fatalf(" Migration failed: %v", err)
	}

	if err := db.Exec(`
		CREATE OR REPLACE FUNCTION prevent_security_audit_mutation() RETURNS trigger AS $$
		BEGIN
			RAISE EXCEPTION 'security audit records are append-only';
		END;
		$$ LANGUAGE plpgsql;
	`).Error; err != nil {
		log.Fatalf("Security audit immutability function migration failed: %v", err)
	}
	if err := db.Exec(`DROP TRIGGER IF EXISTS security_audit_append_only ON security_audit_events`).Error; err != nil {
		log.Fatalf("Security audit trigger migration failed: %v", err)
	}
	if err := db.Exec(`
		CREATE TRIGGER security_audit_append_only
		BEFORE UPDATE OR DELETE ON security_audit_events
		FOR EACH ROW EXECUTE FUNCTION prevent_security_audit_mutation();
	`).Error; err != nil {
		log.Fatalf("Security audit immutability migration failed: %v", err)
	}

	if err := createWorkspacesForExistingUsers(db); err != nil {
		log.Fatalf("Workspace migration failed: %v", err)
	}

	log.Println("Database migrations completed")
}

func createWorkspacesForExistingUsers(db *gorm.DB) error {
	var users []models.User
	if err := db.Where(
		"NOT EXISTS (SELECT 1 FROM tenant_memberships WHERE tenant_memberships.user_id = users.id)",
	).Find(&users).Error; err != nil {
		return err
	}

	for _, user := range users {
		if err := db.Transaction(func(tx *gorm.DB) error {
			tenant := models.Tenant{
				Name: fmt.Sprintf("%s's workspace", user.FullName),
				Slug: fmt.Sprintf("tenant-%d", user.ID),
			}
			if err := tx.Create(&tenant).Error; err != nil {
				return err
			}
			project := models.Project{TenantID: tenant.ID, Name: "Default", Slug: "default"}
			if err := tx.Create(&project).Error; err != nil {
				return err
			}
			if err := tx.Create(&models.TenantMembership{
				TenantID: tenant.ID, UserID: user.ID, Role: "owner",
			}).Error; err != nil {
				return err
			}
			return tx.Create(&models.ProjectMembership{
				ProjectID: project.ID, UserID: user.ID, Role: "owner",
			}).Error
		}); err != nil {
			return err
		}
	}
	return nil
}
