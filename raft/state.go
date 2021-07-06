package raft

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	ll "github.com/emirpasic/gods/lists/singlylinkedlist"
	"github.com/joho/godotenv"

	"github.com/Ishan27g/file-index-lookup-engine/parser"

	comms "github.com/Ishan27g/file-index-lookup-engine/raft/grpc"
)

var timers int
var randomInt = func() int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(1000)
}

const (
	electionTimeout = 20000
	LEADER          = iota + 1
	CANDIDATE
	FOLLOWER
)

var inst string

type Raft interface {
	Details()
	GetState() int
	GetTerm() int
	Run(inst string)
	SendLookup(msg string) *[]parser.SFile
	SetParser(*parser.Parser)
	Quit()
}
type State struct {
	state          int
	termCount      int
	grpcPortString string
	gClient        *ll.List
	http           *HttpSrv
	gServer        *GrpcSvr
	done           chan bool
	pool           sync.Pool
	parser         *parser.Parser
}

func (s *State) SetParser(p *parser.Parser) {
	s.parser = p
	s.gServer.Logg.SetParser(p)
}

var getLeader = func(s *State) *Client {
	if s.state != FOLLOWER {
		return nil
	}
	c1, _ := s.gClient.Get(1)
	switch c := c1.(type) {
	case *Client:
		return c
	}
	return nil
}

func (s *State) Quit() {
	s.done <- true
	s.http.StopHttp()
	getLeader(s).Close()
	s.gServer.Server.StopGrpc()
}

func (s *State) SetState(st int) {
	s.state = st
	s.http.raftState = st
}
func (s *State) GetState() int {
	return s.state
}
func (s *State) GetTerm() int {
	return s.termCount
}
func (s *State) log(i ...interface{}) {

	fmt.Print("-[RAFT]- ")
	for x := 0; x < len(i); x++ {
		fmt.Print(" ", i[x])
	}
	fmt.Println("")

}
func (s *State) Details() {
	s.log("State - ", s.state, " Term - ", s.termCount)
	if s.state == FOLLOWER {
		s1, _ := s.gClient.Get(0)
		s.log("Leader - grpc - ", s1)
	} else if s.state == LEADER {
		s.log("Followers - grpc", s.gServer.Logg.followers.String())
	}
}

var loadGrpcConfig = func() *ll.List {
	err := godotenv.Load(".env")
	if err != nil {
		os.Exit(1)
	}
	instance, _ := strconv.Atoi(inst)
	possiblePeers := ll.New()
	var wg sync.WaitGroup
	for i := 1; i <= 5; i++ {
		if instance != i {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				port := "service" + strconv.Itoa(i) + ":" + os.Getenv("GRPC-"+strconv.Itoa(i))
				//	fmt.Println("CONNECTING TO ", port)
				if client := StartGrpcClient(port); client != nil {
					possiblePeers.Add(client)
				}
			}(i)
		}
	}
	wg.Wait()
	return possiblePeers
}

// NewTimer sends data to returned channel on timeout. Use quit channel to
func NewTimer(timeout int) (timedOut <-chan bool, quits chan<- bool) {
	quit := make(chan bool)
	timed := make(chan bool)
	randomDelay := time.Duration(timeout + randomInt())
	timers++
	go func() {
		select {
		case <-quit:
			timers--
			return
		case <-time.After(randomDelay * time.Millisecond):
			timers--
			close(timed)
		}
	}()
	return timed, quit
}

var LoadPort = func(instance string) string {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	port := os.Getenv("HTTP-" + instance)
	return port
}

func Init(instance string, bootStrap bool) Raft {
	port := LoadPort(instance)
	inst = instance

	this := &State{
		state:          FOLLOWER,
		termCount:      0,
		gServer:        nil,
		gClient:        ll.New(),
		grpcPortString: fmt.Sprintf("service%s:%d%s", inst, 900, inst),
		http:           StartHttp(port, inst),
		done:           make(chan bool),
		pool:           sync.Pool{New: NewSf},
		parser:         nil,
	}
	this.gServer = InitGrpc(this.grpcPortString)

	go func() {
		for {
			select {
			case <-this.done:
				return
			case x := <-this.gServer.Server.UpdateLeaderGrpcPort:
				this.gClient.Clear()
				this.gClient.Add(x)
			case x := <-this.gServer.Server.UpdateTermCount:
				this.termCount = x
			case x := <-this.http.UpdateLeaderGrpcPort:
				this.gClient.Clear()
				this.gClient.Add(x)
			case x := <-this.http.UpdateTermCount:
				this.termCount = x
			}
		}
	}()
	if bootStrap {
		this.becomeLeader()
		go func() {
			if this.StartGrpcElection() { // elected leader via grpc
				this.gServer.Server.raftState = this.state
				this.gServer.Server.termCount = this.termCount
			}
		}()
	}
	return this
}
func (s *State) UpdateGlobalState(state, term int) {
	s.state = state
	s.gServer.Server.raftState = state
	s.http.termCount = term
	s.gServer.Server.termCount = term
	if state == LEADER {
		s.becomeLeader()
	}
}
func (s *State) becomeLeader() {
	s.state = LEADER
	s.http.leaderGrpcPort = s.grpcPortString
	s.gServer.Logg.UpdatePeers(loadGrpcConfig())
}
func (s *State) StartGrpcElection() bool {
	if s.state != FOLLOWER {
		return false
	}
	var wg sync.WaitGroup
	votes := 0
	s.state = CANDIDATE
	c := loadGrpcConfig() // max 100*2 ms in background
	s.termCount++
	s.log("Starting election for term - ", s.termCount, " & ", c.Size(), " votes")
	for it := c.Iterator(); it.Next(); {
		switch c := it.Value().(type) {
		case *Client:
			wg.Add(1)
			go func(termCount int) {
				defer wg.Done()
				if c.SendVoteReq(termCount, s.grpcPortString) {
					votes++
				}
			}(s.termCount)
		}
	}
	wg.Wait()
	fmt.Println(votes, c.Size())
	if votes == c.Size() {
		s.state = LEADER
		s.log("ELECTED LEADER with ", votes, "votes")
		return true
	}
	s.state = FOLLOWER
	return false
}
func NewSf() interface{} {
	return &comms.QueryResponse{SFile: nil}
}

func (s *State) SendLookup(word string) *[]parser.SFile { //nolint:cyclop    // lookup
	var rspSfList []parser.SFile
	if s.state == LEADER {
		if s.gClient.Size() != 1 {
			return nil
		}
		c := s.gClient
		votes := 0
		for it := c.Iterator(); it.Next(); {
			switch c := it.Value().(type) {
			case *Client:
				rsp := c.SendLookupQuery(word, s.grpcPortString)
				if rsp != nil {
					for _, r := range rsp.SFile {
						rspSfList = append(rspSfList, parser.SFile{
							Filename:  r.FileName,
							FileLoc:   r.FileLoc,
							LineNum:   r.LineLoc,
							WordIndex: r.WordLoc,
						})
					}
					votes++
				} else {
					fmt.Println("Peer send nil response")
				}
			}
		}
		fmt.Println(votes, c.Size())
		if votes == c.Size() {
			return &rspSfList
		} else {
			return nil
		}
	} else if s.state == FOLLOWER {
		if s.gClient.Size() != 1 {
			return &rspSfList
		}
		c1, _ := s.gClient.Get(0) // leader
		// fmt.Println("Current leader is - ", c1)
		client := StartGrpcClient(fmt.Sprintf("%s", c1))
		rsp := client.SendLookupQuery(word, s.grpcPortString)
		if rsp != nil {
			for _, r := range rsp.SFile {
				rspSfList = append(rspSfList, parser.SFile{
					Filename:  r.FileName,
					FileLoc:   r.FileLoc,
					LineNum:   r.LineLoc,
					WordIndex: r.WordLoc,
				})
			}
		} else {
			return nil
		}
	}
	return &rspSfList
}
func (s *State) Run(inst string) {
	s.http.UpdatePeerList(inst) // akin to discovery
	s.http.raftState = s.state
	if s.GetState() == LEADER {
		heartTimeout, _ := NewTimer(electionTimeout / 3)
		if <-heartTimeout; true {
			s.http.SendHeartBeat(s.termCount, s.grpcPortString) // http heartbeat
			s.gServer.Logg.UpdatePeers(loadGrpcConfig())
		}
	} else if s.GetState() == FOLLOWER {
		heartTimeout, quit := NewTimer(electionTimeout)
		select {
		case <-s.http.HeartBeatReceived:
			quit <- true
		case <-heartTimeout:
			if s.StartGrpcElection() { // elected leader via grpc
				s.UpdateGlobalState(s.state, s.termCount)
			}
		}
	} else {
		// state = CANDIDATE
		time.Sleep(electionTimeout / 5 * time.Millisecond)
	}
}
