package router

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	
	"ishan/FSI/redisClient"
	
	"ishan/FSI/parser"
)

var service Service
type Service struct {
	hostname string
	port string
	parser *parser.Parser
	rclient redisClient.Rdb
}
func pong(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{"ok":"Pong"})
}
func NewGinServer(p *parser.Parser, inst string) (*gin.Engine, string){
	_ = godotenv.Load("./router/.env")
	basePort,_ := strconv.Atoi(os.Getenv("HTTP-PORT"))
	i, _ := strconv.Atoi(inst)
	host, _ := os.Hostname()
	
	service = Service{
		parser: p,
		rclient: redisClient.Init(inst),
		port:  ":" + strconv.Itoa(basePort+i),
		hostname: host,
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
	
	log.Println("Configured for - ",host, service.port )
	
	return r, service.port
}

func (p *Service)search(c *gin.Context) {
	word := c.Query("word")
	if word == ""{
		c.JSON(http.StatusBadRequest, "Missing params - word")
		return
	}
	sfList := p.parser.Find(word)
	_ = parser.Stringify(sfList)
	r := p.rclient.Get(sfList[0].Filename)
	fmt.Println(r, len(r))
	c.JSON(http.StatusOK, gin.H{"found":sfList[0].Filename})
}

func (p *Service)fileUpload(c *gin.Context) {
	form, _ := c.MultipartForm()
	files := form.File["files"]
	for _, file := range files {
		dst := path.Join("./sample_files/", file.Filename)
		err := c.SaveUploadedFile(file, dst)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusExpectationFailed, gin.H{"Not Uploaded" : "no"})
			return
		}
		go func() {
			if p.parser.AddFile(dst) {
				fmt.Println("Saving to redis", dst, p.hostname+p.port)
				p.rclient.Add(dst, p.hostname+p.port)
			}
		}()
	}
	c.JSON(http.StatusOK, gin.H{"Uploaded" : fmt.Sprintf("%d files!", len(files))})
}