package controllers

import (
	"atonixcorp/api/internal/models"
	"atonixcorp/api/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type BlogController struct {
	service services.BlogService
}

func NewBlogController(service services.BlogService) *BlogController {
	return &BlogController{service: service}
}

func (c *BlogController) Create(ctx *gin.Context) {
	var post models.BlogPost

	if err := ctx.BindJSON(&post); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	if err := c.service.Create(&post); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create blog post"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Blog created"})
}

func (c *BlogController) Update(ctx *gin.Context) {
	var post models.BlogPost

	if err := ctx.BindJSON(&post); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	if err := c.service.Update(&post); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update blog post"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Blog updated"})
}

func (c *BlogController) Delete(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid blog ID"})
		return
	}

	if err := c.service.Delete(uint(id)); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete blog post"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Blog deleted"})
}

func (c *BlogController) GetByID(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.Atoi(idParam)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid blog ID"})
		return
	}

	post, err := c.service.GetByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Blog not found"})
		return
	}

	ctx.JSON(http.StatusOK, post)
}

func (c *BlogController) GetAll(ctx *gin.Context) {
	posts, err := c.service.GetAll()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve blog posts"})
		return
	}

	ctx.JSON(http.StatusOK, posts)
}
