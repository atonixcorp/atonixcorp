package controllers

import (
	"atonixcorp/api/internal/models"
	"atonixcorp/api/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ContactController struct {
	service services.ContactService
}

func NewContactController(service services.ContactService) *ContactController {
	return &ContactController{service: service}
}

func (c *ContactController) Submit(ctx *gin.Context) {
	var msg models.ContactMessage

	if err := ctx.BindJSON(&msg); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	if err := c.service.Submit(&msg); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save message"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Message received"})
}

func (c *ContactController) GetAll(ctx *gin.Context) {
	messages, err := c.service.GetAll()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve messages"})
		return
	}

	ctx.JSON(http.StatusOK, messages)
}
