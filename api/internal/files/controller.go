package files

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Controller struct{ service *Service }

func NewController(service *Service) *Controller { return &Controller{service: service} }

func userID(c *gin.Context) uint {
	value, _ := c.Get("user_id")
	id, _ := value.(uint)
	return id
}

func (c *Controller) Upload(ctx *gin.Context) {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, MaxUploadBytes+(1<<20))
	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid or oversized multipart upload"})
		return
	}
	defer file.Close()
	if ctx.Request.MultipartForm != nil {
		defer ctx.Request.MultipartForm.RemoveAll()
	}
	tenantID, err := strconv.ParseUint(ctx.PostForm("tenant_id"), 10, 32)
	if err != nil || tenantID == 0 || userID(ctx) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "valid tenant_id is required"})
		return
	}
	obj, err := c.service.Upload(ctx.Request.Context(), userID(ctx), uint(tenantID), header.Filename, header.Header.Get("Content-Type"), file)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, obj)
}

func (c *Controller) List(ctx *gin.Context) {
	tenantID, err := strconv.ParseUint(ctx.Query("tenant_id"), 10, 32)
	if err != nil || tenantID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "valid tenant_id is required"})
		return
	}
	limit, offset := 50, 0
	if value := ctx.Query("limit"); value != "" {
		limit, err = strconv.Atoi(value)
		if err != nil {
			writeError(ctx, ErrInvalid)
			return
		}
	}
	if value := ctx.Query("offset"); value != "" {
		offset, err = strconv.Atoi(value)
		if err != nil {
			writeError(ctx, ErrInvalid)
			return
		}
	}
	objects, err := c.service.List(ctx.Request.Context(), userID(ctx), uint(tenantID), limit, offset)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, objects)
}

func (c *Controller) AuthorizeUpload(ctx *gin.Context) {
	var request UploadRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid upload request"})
		return
	}
	result, err := c.service.AuthorizeUpload(ctx.Request.Context(), userID(ctx), request)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusCreated, result)
}

func (c *Controller) CompleteUpload(ctx *gin.Context) {
	obj, err := c.service.CompleteUpload(ctx.Request.Context(), userID(ctx), ctx.Param("id"))
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, obj)
}

func (c *Controller) Download(ctx *gin.Context) {
	obj, body, err := c.service.Download(ctx.Request.Context(), userID(ctx), ctx.Param("id"))
	if err != nil {
		writeError(ctx, err)
		return
	}
	defer body.Close()
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", obj.OriginalFilename))
	ctx.Header("X-Content-Type-Options", "nosniff")
	ctx.Header("Cache-Control", "private, no-store")
	ctx.Header("Content-Length", strconv.FormatInt(obj.SizeBytes, 10))
	ctx.DataFromReader(http.StatusOK, obj.SizeBytes, obj.ContentType, body, nil)
}

func (c *Controller) Presign(ctx *gin.Context) {
	url, err := c.service.Presign(ctx.Request.Context(), userID(ctx), ctx.Param("id"))
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, gin.H{"url": url, "expires_in": 300})
}

func (c *Controller) Delete(ctx *gin.Context) {
	if err := c.service.Delete(ctx.Request.Context(), userID(ctx), ctx.Param("id")); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (c *Controller) Health(ctx *gin.Context) {
	timeout, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()
	if err := c.service.Health(timeout); err != nil {
		log.Printf("object storage health: %v", err)
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func writeError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalid):
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid file, content type, or size"})
	case errors.Is(err, ErrNotFound):
		ctx.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
	case errors.Is(err, ErrUnavailable):
		log.Printf("object storage request failed: %v", err)
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "object storage unavailable"})
	default:
		if !errors.Is(err, io.EOF) {
			log.Printf("file request failed: %v", err)
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "file operation failed"})
	}
}
