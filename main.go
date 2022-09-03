package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type sumRequest struct {
	A int `json:"a" binding:"required"`
	B int `json:"b" binding:"required"`
}

func sum(a, b int) int {
	return a + b
}

func sumHandler(c *gin.Context) {
	var req sumRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": err.Error(),
		})
		return
	}

	result := sum(req.A, req.B)

	c.JSON(http.StatusOK, gin.H{
		"result": result,
	})
}

func main() {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.POST("/sum", sumHandler)

	r.Run()
}
