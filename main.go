package main

import (
	"flag"
	"fmt"
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
	
	raft := comms.NewRaftService(bs, inst)
	
	p := parser.Start(fmt.Sprintf(":%d%s", 900, inst))
	rservice := router.NewGinServer(p, inst)
	raft.SetParser(p)
	for  {
		select {
		case word:= <- rservice.LookupChan:
			raft.ForwardLog <- word
			sfList := <- raft.ResponseSfList
			rservice.ResponseSfiles <- sfList
		}
	}
}
