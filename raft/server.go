package raft

import (
	"net"
	"os"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	comms "github.com/Ishan27g/file-index-lookup-engine/raft/grpc"
)

var stop chan bool

func logIt(v ...interface{}) {
	/*
		fmt.Print("-[GRPC]- ")
		for x := 0; x < len(v); x++ {
			fmt.Print(" ", v[x])
		}
		fmt.Println("")
	*/
}

type GrpcSvr struct {
	Server *Server
	Logg   *LData
}
type Server struct {
	portString           string
	raftState            int
	termCount            int
	leaderPort           string
	UpdateTermCount      chan int
	UpdateLeaderGrpcPort chan string
}

func (s *Server) RequestVotes(ctx context.Context, term *comms.Term) (*comms.Vote, error) {
	if s.raftState != FOLLOWER {
		return &comms.Vote{Elected: false}, nil
	}
	// if new one from current leader
	if strings.Compare(s.leaderPort, term.LeaderPort) == 0 &&
		s.termCount < int(term.TermCount) {
		s.termCount = int(term.TermCount)
		s.leaderPort = term.LeaderPort
		s.UpdateLeaderGrpcPort <- s.leaderPort
		s.UpdateTermCount <- s.termCount
		return &comms.Vote{Elected: true}, nil
	} else
	// if from new leader
	if s.termCount < int(term.TermCount) {
		s.termCount = int(term.TermCount)
		s.leaderPort = term.LeaderPort
		s.UpdateLeaderGrpcPort <- s.leaderPort
		s.UpdateTermCount <- s.termCount
		return &comms.Vote{Elected: true}, nil
	}
	return &comms.Vote{Elected: false}, nil
}

func (s *Server) StopGrpc() {
	stop <- true
	close(s.UpdateTermCount)
	close(s.UpdateLeaderGrpcPort)
}
func InitGrpc(port string) *GrpcSvr {

	stop = make(chan bool)
	logg := NewDataSync(port)
	serv := Server{
		portString:           port,
		raftState:            FOLLOWER,
		termCount:            0,
		leaderPort:           "",
		UpdateTermCount:      make(chan int),
		UpdateLeaderGrpcPort: make(chan string),
	}
	lis, err := net.Listen("tcp", serv.portString)
	if err != nil {
		logIt("Exiting - failed to listen: %v", err)
		os.Exit(1)
	}
	grpcServer := grpc.NewServer()
	comms.RegisterRaftServer(grpcServer, &serv)
	comms.RegisterLookupServer(grpcServer, logg)
	logIt("Listening on ", serv.portString)
	go func() {
		if err := grpcServer.Serve(lis); err != nil { // blocking
			logIt("failed to serve: %s", err)
			os.Exit(1)
		}
	}()
	go func() {
		<-stop
		grpcServer.GracefulStop()
	}()

	return &GrpcSvr{
		Server: &serv,
		Logg:   logg,
	}
}
