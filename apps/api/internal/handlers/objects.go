package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func CreateObject (c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "Not implemented yet",
	})
}

func GetObject(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "Not implemented yet",
	})
}