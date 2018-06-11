package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func handleIndex(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}
