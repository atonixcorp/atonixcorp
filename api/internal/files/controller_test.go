package files

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"atonixcorp/api/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFileHTTPWorkflow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.StorageObject{}, &models.ObjectAuditEvent{}, &models.TenantMembership{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.TenantMembership{TenantID: 1, UserID: 1, Role: "owner"}).Error; err != nil {
		t.Fatal(err)
	}
	controller := NewController(NewService(db, &fakeProvider{data: map[string][]byte{}}))
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if c.GetHeader("Test-User") == "other" {
			c.Set("user_id", uint(2))
		} else {
			c.Set("user_id", uint(1))
		}
		c.Next()
	})
	router.POST("/files/upload", controller.Upload)
	router.GET("/files/:id/download", controller.Download)
	router.GET("/files/:id/download-url", controller.Presign)
	router.DELETE("/files/:id", controller.Delete)
	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)
	if err := writer.WriteField("tenant_id", "1"); err != nil {
		t.Fatal(err)
	}
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="file"; filename="report.txt"`)
	header.Set("Content-Type", "text/plain")
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(part, "hello"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/files/upload", &payload)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("upload: %d %s", res.Code, res.Body.String())
	}
	var obj models.StorageObject
	if err := json.Unmarshal(res.Body.Bytes(), &obj); err != nil || obj.ID == "" {
		t.Fatalf("upload response: %s %v", res.Body.String(), err)
	}
	req = httptest.NewRequest(http.MethodGet, "/files/"+obj.ID+"/download", nil)
	req.Header.Set("Test-User", "other")
	res = httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("unauthorized download: %d", res.Code)
	}
	req = httptest.NewRequest(http.MethodGet, "/files/"+obj.ID+"/download", nil)
	res = httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK || res.Body.String() != "hello" || res.Header().Get("Content-Disposition") != `attachment; filename="report.txt"` {
		t.Fatalf("download: %d %s", res.Code, res.Body.String())
	}
	req = httptest.NewRequest(http.MethodDelete, "/files/"+obj.ID, nil)
	res = httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", res.Code, res.Body.String())
	}
}
