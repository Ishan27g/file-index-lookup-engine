package main
import(
	"flag"
	"fmt"
	"log"
	"os"
	
	"ishan/FSI/comms"
	parser "ishan/FSI/parser"
	"ishan/FSI/router"
)

func load() (bool, string) {
	var bootstrap bool
	var inst string
	flag.BoolVar(&bootstrap, "bootstrap", false, "bootstrap a leader for first run")
	flag.StringVar(&inst, "instance", "", "unique instance(1-5)")
	flag.Parse()
	if inst == ""{
		fmt.Println("Invalid options")
		fmt.Println("Usage - ./main -instance=[1..5] -bootstrap=true")
		fmt.Println("Usage - ./main -instance=[1..5] -bootstrap=false")
		os.Exit(1)
	}
	return bootstrap, inst //nolint:govet
}
func main(){
	bs, inst := load()
	
	_ = comms.NewRaftService(bs, inst)
	
	p := parser.Start(inst)
	r,port := router.NewGinServer(p, inst)
	
	err := r.Run(port)
	if err != nil {
		log.Fatalf(err.Error(),err)
	}
}
