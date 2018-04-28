package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"io/ioutil"
	"os"
	"path"
	"bytes"
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

func handleIndex(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}

func addFile(f os.FileInfo, parent file) (file, error) {
	if f.IsDir() {
		files, err := ioutil.ReadDir(path.Join(path.Join(parent.Path...), f.Name()))
		if err != nil {
			return file{}, nil
		}
		dir := file{Name: f.Name(), Type: dirFile, Path: append(parent.Path, f.Name()), Files: []file{}}
		for _, f := range files {
			nf, err := addFile(f, dir)
			if err != nil {
				return file{}, nil
			}
			dir.Files = append(dir.Files, nf)
		}
		return dir, nil
	} else {
		return file{
			Type:  fileFile,
			Name:  f.Name(),
			Path:  parent.Path,
			Files: nil,
		}, nil
	}
}

func handleApiFiles(c *gin.Context) {
	files, err := ioutil.ReadDir(".")
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	root := file{Type: dirFile, Name: "/", Path: []string{}, Files: []file{}}
	for _, f := range files {
		nf, err := addFile(f, root)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		root.Files = append(root.Files, nf)
	}

	c.IndentedJSON(200, root)
	return
}

func handleApiFile(c *gin.Context) {
	filePath := c.Param("path")

	f, err := ioutil.ReadFile(filePath[1:])
	if err != nil {
		c.AbortWithError(404, err)
	}

	c.JSON(200, struct {
		Data string `json:"data"`
	}{bytes.NewBuffer(f).String()})
	return
}
func handleApiSave(c *gin.Context) {
	filePath := c.Param("path")

	f, err := ioutil.ReadFile(filePath[1:])
	if err != nil {
		c.AbortWithError(404, err)
	}

	c.JSON(200, struct {
		Data string `json:"data"`
	}{bytes.NewBuffer(f).String()})
	return
}

func main() {

	log.Println("Setting up routes...")
	router := gin.Default()

	router.LoadHTMLFiles("web/index.html")
	router.Static("/static/", "./web/static")

	// Main query interface.
	router.GET("/", handleIndex)
	router.GET("/api/files", handleApiFiles)
	router.GET("/api/file/*path", handleApiFile)
	router.GET("/api/save/*path", handleApiSave)

	log.Println("let's go!")
	log.Fatal(http.ListenAndServe(":5862", router))
}
