package router

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/Ishan27g/file-index-lookup-engine/parser"
)

var service Service

type Service struct {
	hostname       string
	port           string
	parser         *parser.Parser
	LookupChan     chan string
	ResponseSfiles chan *[]parser.SFile
}

func pong(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{"ok": "Pong"})
}
func NewGinServer(p *parser.Parser, inst string) *Service {
	_ = godotenv.Load(".env")
	basePort, _ := strconv.Atoi(os.Getenv("HTTP-PORT"))
	_, _ = strconv.Atoi(inst)
	host, _ := os.Hostname()

	service = Service{
		parser:         p,
		port:           ":" + strconv.Itoa(basePort), // todo +i only w/o conf.d
		hostname:       host,
		LookupChan:     make(chan string),
		ResponseSfiles: make(chan *[]parser.SFile),
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.MaxMultipartMemory = 8
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	s := r.Group("/service")
	{
		s.GET("/ping", pong)
		s.POST("/upload", service.fileUpload)
		s.GET("/search", service.search)
	}

	log.Println("Configured for - ", host, service.port)
	go func() {
		err := r.Run(service.port)
		if err != nil {
			log.Fatalf(err.Error(), err)
		}
	}()
	return &service
}

func (p *Service) search(c *gin.Context) {
	word := c.Query("word")
	if word == "" {
		c.JSON(http.StatusBadRequest, "Missing params - word")
		return
	}
	var sFiles *[]parser.SFile
	done := make(chan bool)
	go func() {
		p.LookupChan <- word
		sFiles = <-p.ResponseSfiles
		done <- true
	}()
	timeout, _ := strconv.Atoi(os.Getenv("GRPC-Time-Out"))
	sfList := p.parser.Find(word)
	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		fmt.Println("----------timing out---------")
		c.JSON(http.StatusOK, gin.H{"local": sfList})
	case <-done:
		c.JSON(http.StatusOK, gin.H{"local": sfList, "remote": sFiles})
	}
}

func (p *Service) fileUpload(c *gin.Context) {
	form, _ := c.MultipartForm()
	files := form.File["files"]
	for _, file := range files {
		dst := path.Join("./sample_files/", file.Filename)
		err := c.SaveUploadedFile(file, dst)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusExpectationFailed, gin.H{"Not Uploaded": "no"})
			return
		}

		go func() {
			if p.parser.AddFile(dst) {
				// fmt.Println("Saving to redis", dst, p.hostname+p.port)
				// p.rclient.Add(dst, p.hostname+p.port)
			}
		}()
	}
	c.JSON(http.StatusOK, gin.H{"Uploaded": fmt.Sprintf("%d files!", len(files))})
}
