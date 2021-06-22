package raft

import (
	"ishan/FSI/parser"
)

type RaftService struct {
	ForwardLog chan string // search for this word
	done chan bool
	raft Raft
	ResponseSfList chan *[]parser.SFile
}

func (rs *RaftService)Quit(){
	rs.done <- true
	close(rs.ForwardLog)
	rs.raft.Quit()
}
func NewRaftService(bootstrap bool, instance string) *RaftService{
	// http to reach consensus (for heartbeat & initiating election on timeout)
	// grpc to send updates (both ways -> server always on, client created per request)

	rs := RaftService{
		ForwardLog: make(chan string, 5),
		ResponseSfList: make(chan *[]parser.SFile, 5),
		done:       make(chan bool),
		raft :      Init(instance, bootstrap),
	}
	go func() {
		for {
			rs.raft.Run(inst)
			// rs.raft.Details()
			select {
			case <-rs.done:
				return
			case logMsg := <-rs.ForwardLog:
				rs.ResponseSfList <- rs.raft.SendLookup(logMsg) // queue back in case of network partition?
			default:
				continue
			}
		}
	}()

	return &rs
}
func (rs *RaftService)GetState()int{
	return rs.raft.GetState()
}

func (rs *RaftService) SetParser(p *parser.Parser) {
	rs.raft.SetParser(p)
}