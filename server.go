package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"github.com/hscells/groove"
	"fmt"
	"io"
	"os"
)

const (
	dirFile  = iota
	fileFile
)

type file struct {
	Type  int      `json:"type"`
	Name  string   `json:"name"`
	Path  []string `json:"path"`
	Files []file   `json:"files"`
}

type pipelineResult struct {
	Type   groove.ResultType `json:"type"`
	Result string            `json:"pipelineResult"`
}

func main() {

	lf, err := os.OpenFile("web/static/log", os.O_WRONLY|os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	lf.Truncate(0)

	router := gin.Default()

	router.LoadHTMLFiles("web/index.html")
	router.Static("/static/", "./web/static")

	// Main query interface.
	router.GET("/", handleIndex)
	router.GET("/api/files", handleApiFiles)
	router.GET("/api/file/*path", handleApiFile)
	router.POST("/api/save/*path", handleApiSave)
	router.POST("/api/run", handleApiRun)

	mw := io.MultiWriter(lf, os.Stdout)
	log.SetOutput(mw)

	fmt.Print(`

 .d88888b.                                     888          888      
d88P" "Y88b                                    888          888      
888     888                                    888          888      
888     888 888  888  .d88b.  888d888 888  888 888  8888b.  88888b.  
888     888 888  888 d8P  Y8b 888P"   888  888 888     "88b 888 "88b 
888 Y8b 888 888  888 88888888 888     888  888 888 .d888888 888  888 
Y88b.Y8b88P Y88b 888 Y8b.     888     Y88b 888 888 888  888 888 d88P 
 "Y888888"   "Y88888  "Y8888  888      "Y88888 888 "Y888888 88888P"  
       Y8b                                 888                       
                                      Y8b d88P                       
                                       "Y88P"

 Harry Scells 2018

`)
	log.Fatal(http.ListenAndServe(":5862", router))
}
