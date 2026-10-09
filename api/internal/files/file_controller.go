package files

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type FileController struct {
	service *FileService
}

func NewFileController(service *FileService) *FileController {
	return &FileController{service}
}

func (c *FileController) Upload(ctx *gin.Context) {
	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	defer file.Close()

	saved, err := c.service.Upload(file, header)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, saved)
}

func (c *FileController) Download(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	file, err := c.service.repo.GetByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	ctx.File(file.FilePath)
}
