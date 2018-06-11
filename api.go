package main

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/hscells/boogie"
	"github.com/hscells/groove"
	"github.com/hscells/transmute/backend"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// upgrader is a struct that upgrades a web socket.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var (
	writeWait = 2000 * time.Millisecond
)

type statistics struct {
	CPU    []float64
	Memory float64
}

func wsStatistics(ws *websocket.Conn) {

	writeTicker := time.NewTicker(writeWait)

	// defer closing of web socket
	defer func() {
		writeTicker.Stop()
		ws.Close()
	}()
	// create a new go routine for the web socket
	for {
		select {
		case <-writeTicker.C:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			v, _ := mem.VirtualMemory()
			c, _ := cpu.Percent(0, true)
			if err := ws.WriteJSON(statistics{Memory: v.UsedPercent, CPU: c}); err != nil {
				log.Println("writing", err)
				return
			}
		}
	}
}

func handleWsStatistics(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}

	go wsStatistics(ws)
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
		c.String(http.StatusNotFound, err.Error())
		return
	}

	root := file{Type: dirFile, Name: "/", Path: []string{}, Files: []file{}}
	for _, f := range files {
		nf, err := addFile(f, root)
		if err != nil {
			c.String(http.StatusNotFound, err.Error())
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
		c.String(http.StatusNotFound, err.Error())
		return
	}

	c.JSON(200, struct {
		Data string `json:"data"`
	}{bytes.NewBuffer(f).String()})
	return
}

func handleApiSave(c *gin.Context) {
	filePath := c.Param("path")
	content := c.PostForm("content")

	var p string
	if len(filePath) > 0 {
		p = filePath[1:]
	} else {
		c.String(http.StatusInternalServerError, "no path specified")
		return
	}

	err := ioutil.WriteFile(p, []byte(content), 0644)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.Status(200)
	return
}

func handleApiRun(c *gin.Context) {
	queryPath := c.PostForm("path")
	pipelineData := c.PostForm("pipeline")

	var dsl boogie.Pipeline
	err := json.Unmarshal([]byte(pipelineData), &dsl)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	pipeline, err := boogie.CreatePipeline(dsl)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	evaluations := make([]string, len(dsl.Evaluations))

	trecEvalBuff := bytes.NewBuffer([]byte{})

	pipelineChan := make(chan groove.PipelineResult)

	var trecEvalFile *os.File
	if len(dsl.Output.Trec.Output) > 0 {
		trecEvalFile, err = os.OpenFile(dsl.Output.Trec.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalln(err)
		}
		trecEvalFile.Truncate(0)
		trecEvalFile.Seek(0, 0)
		defer trecEvalFile.Close()
	}

	go pipeline.Execute(queryPath, pipelineChan)
	for {
		result := <-pipelineChan
		if result.Type == groove.Done {
			break
		}
		switch result.Type {
		case groove.Measurement:
			// Process the measurement outputs.
			for i, formatter := range dsl.Output.Measurements {
				err := ioutil.WriteFile(formatter.Filename, bytes.NewBufferString(result.Measurements[i]).Bytes(), 0644)
				if err != nil {
					c.String(http.StatusInternalServerError, err.Error())
					return
				}
			}
		case groove.Transformation:
			// Output the transformed queries
			if len(dsl.Transformations.Output) > 0 {
				s, err := backend.NewCQRQuery(result.Transformation.Transformation).StringPretty()
				if err != nil {
					log.Fatalln(err)
				}
				q := bytes.NewBufferString(s).Bytes()
				err = ioutil.WriteFile(filepath.Join(pipeline.Transformations.Output, result.Transformation.Name), q, 0644)
				if err != nil {
					c.String(http.StatusInternalServerError, err.Error())
					return
				}
			}
		case groove.Evaluation:
			for i, e := range result.Evaluations {
				evaluations[i] = e
			}
		case groove.TrecResult:
			if result.TrecResults != nil && len(*result.TrecResults) > 0 {
				l := make([]string, len(*result.TrecResults))
				for i, r := range *result.TrecResults {
					l[i] = r.String()
				}
				trecEvalBuff.Write([]byte(strings.Join(l, "\n") + "\n"))
				trecEvalFile.Write([]byte(strings.Join(l, "\n") + "\n"))
				result.TrecResults = nil
			}
			trecEvalBuff.Truncate(0)
		case groove.Error:
			if len(result.Topic) > 0 {
				log.Printf("an error occurred in topic %v", result.Topic)
			} else {
				log.Println("an error occurred")
			}
			c.AbortWithError(500, err)
			return
		}
	}
	// Process the evaluation outputs.
	for i, formatter := range dsl.Output.Evaluations.Measurements {
		err := ioutil.WriteFile(formatter.Filename, bytes.NewBufferString(evaluations[i]).Bytes(), 0644)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
	}
	c.Status(200)
	return
}
