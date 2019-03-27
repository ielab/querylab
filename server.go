package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gobuffalo/packr/v2"
	"io"
	"log"
	"net/http"
	"os"
)

const (
	dirFile = iota
	fileFile
)

type file struct {
	Type  int      `json:"type"`
	Name  string   `json:"name"`
	Path  []string `json:"path"`
	Files []file   `json:"files"`
}

func main() {

	lf, err := os.OpenFile("querylab.log", os.O_WRONLY|os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalln(err)
	}

	staticBox := packr.New("static", "./web/static")
	webBox := packr.New("web", "./web")

	router := gin.Default()

	router.GET("/static/*any", gin.WrapH(http.StripPrefix("/static/", http.FileServer(staticBox))))

	// Main query interface.
	router.GET("/", func(c *gin.Context) {
		s, err := webBox.Find("index.html")
		if err != nil {
			log.Fatal(err)
		}
		c.Data(http.StatusOK, "text/html", s)
	})

	router.GET("/api/files", handleApiFiles)
	router.GET("/api/file/*path", handleApiFile)
	router.POST("/api/save/*path", handleApiSave)
	router.POST("/api/run", handleApiRun)
	router.GET("/ws/statistics", handleWsStatistics)

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
 https://ielab.io/querylab
`)
	log.Fatal(router.Run(":5862"))
}
