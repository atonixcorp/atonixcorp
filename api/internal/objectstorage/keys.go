package objectstorage

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var keySegment = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,254}$`)
var imageExtension = regexp.MustCompile(`^[A-Za-z0-9]{1,16}$`)

// KeyParts contains trusted, validated path components, not arbitrary object keys.
type KeyParts struct {
	Category      string
	Tenant        string
	User          string
	ID            string
	Filename      string
	ImageCategory string
	UserCategory  string
	Extension     string
	DocumentType  string
	Service       string
	Project       string
	Dataset       string
	Model         string
	Version       string
	Time          time.Time
}

func ObjectKey(p KeyParts) (string, error) {
	valid := func(values ...string) bool {
		for _, value := range values {
			if !keySegment.MatchString(value) || value == "." || value == ".." {
				return false
			}
		}
		return true
	}
	date := p.Time.UTC()
	switch p.Category {
	case "uploads":
		if valid(p.Tenant, p.User, p.ID, p.Filename) {
			return fmt.Sprintf("uploads/%s/%s/%s/%s", p.Tenant, p.User, p.ID, p.Filename), nil
		}
	case "images":
		if valid(p.ImageCategory, p.Tenant, p.ID) && imageExtension.MatchString(p.Extension) {
			return fmt.Sprintf("images/%s/%s/%s.%s", p.ImageCategory, p.Tenant, p.ID, p.Extension), nil
		}
	case "documents":
		if valid(p.Tenant, p.DocumentType, p.ID, p.Filename) && !p.Time.IsZero() {
			return fmt.Sprintf("documents/%s/%s/%04d/%s/%s", p.Tenant, p.DocumentType, date.Year(), p.ID, p.Filename), nil
		}
	case "backups":
		if valid(p.Service, p.Filename) && !p.Time.IsZero() {
			return fmt.Sprintf("backups/%s/%04d/%02d/%s/%s", p.Service, date.Year(), date.Month(), date.Format("20060102T150405.000000000Z"), p.Filename), nil
		}
	case "logs":
		if valid(p.Service, p.Filename) && !p.Time.IsZero() {
			return fmt.Sprintf("logs/%s/%04d/%02d/%02d/%s", p.Service, date.Year(), date.Month(), date.Day(), p.Filename), nil
		}
	case "datasets":
		if valid(p.Project, p.Dataset, p.Version, p.Filename) {
			return fmt.Sprintf("datasets/%s/%s/%s/%s", p.Project, p.Dataset, p.Version, p.Filename), nil
		}
	case "models":
		if valid(p.Project, p.Model, p.Version, p.Filename) {
			return fmt.Sprintf("models/%s/%s/%s/%s", p.Project, p.Model, p.Version, p.Filename), nil
		}
	case "users":
		if valid(p.User, p.UserCategory, p.ID, p.Filename) {
			return fmt.Sprintf("users/%s/%s/%s/%s", p.User, p.UserCategory, p.ID, p.Filename), nil
		}
	}
	return "", errors.New("invalid object key components for category " + strconv.Quote(p.Category))
}
