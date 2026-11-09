package main

import (
	"context"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"atonixcorp/api/internal/database"
	"atonixcorp/api/internal/files"
	"atonixcorp/api/internal/models"
	"atonixcorp/api/internal/objectstorage/minio"

	"gorm.io/gorm"
)

func main() {
	rootPath := flag.String("legacy-root", "", "historical API working directory containing uploads/")
	mappingPath := flag.String("mapping", "", "CSV with legacy_file_id,owner_id,tenant_id")
	actorID := flag.Uint("actor-id", 0, "platform super-admin user ID responsible for this import")
	flag.Parse()
	if *rootPath == "" || *mappingPath == "" || *actorID == 0 {
		log.Fatal("legacy-root, mapping and actor-id are required")
	}
	root, err := os.OpenRoot(*rootPath)
	if err != nil {
		log.Fatalf("open legacy root: %v", err)
	}
	defer root.Close()
	mapping, err := os.Open(*mappingPath)
	if err != nil {
		log.Fatalf("open mapping: %v", err)
	}
	defer mapping.Close()
	db := database.Open()
	provider, err := minio.FromEnvironment()
	if err != nil || provider == nil {
		log.Fatalf("configure object storage: %v", err)
	}
	service := files.NewService(db, provider)
	reader := csv.NewReader(mapping)
	header, err := reader.Read()
	if err != nil || len(header) != 3 || header[0] != "legacy_file_id" || header[1] != "owner_id" || header[2] != "tenant_id" {
		log.Fatal("mapping must start with legacy_file_id,owner_id,tenant_id")
	}
	for line := 2; ; line++ {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			log.Fatalf("mapping line %d: %v", line, err)
		}
		ids := make([]uint, 3)
		for i, value := range row {
			parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 32)
			if err != nil || parsed == 0 {
				log.Fatalf("mapping line %d: invalid ID", line)
			}
			ids[i] = uint(parsed)
		}
		if err := migrateOne(context.Background(), db, service, root, *actorID, ids[0], ids[1], ids[2]); err != nil {
			log.Fatalf("mapping line %d, legacy file %d: %v", line, ids[0], err)
		}
	}
}

func migrateOne(ctx context.Context, db *gorm.DB, service *files.Service, root *os.Root, actorID, legacyID, ownerID, tenantID uint) error {
	var existing models.StorageObject
	err := db.WithContext(ctx).Where("legacy_file_id = ?", legacyID).First(&existing).Error
	if err == nil {
		if existing.Status == "active" && existing.OwnerID == ownerID && existing.TenantID == tenantID {
			log.Printf("legacy file %d already imported as %s", legacyID, existing.ID)
			return nil
		}
		return fmt.Errorf("existing import %s has status %s or a different mapping; reconcile manually", existing.ID, existing.Status)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	var legacy models.File
	if err := db.WithContext(ctx).First(&legacy, legacyID).Error; err != nil {
		return err
	}
	path := filepath.Clean(legacy.FilePath)
	if !filepath.IsLocal(legacy.FilePath) || !strings.HasPrefix(path, "uploads"+string(filepath.Separator)) {
		return errors.New("legacy path is outside uploads/")
	}
	file, err := root.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() != legacy.Size {
		return errors.New("legacy file is not regular or its size does not match metadata")
	}
	obj, err := service.UploadLegacy(ctx, actorID, ownerID, tenantID, legacyID, legacy.FileName, legacy.MimeType, file, info.Size())
	if err != nil {
		return err
	}
	log.Printf("imported legacy file %d as %s; source retained", legacyID, obj.ID)
	return nil
}
