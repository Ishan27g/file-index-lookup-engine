package comms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	
	ll "github.com/emirpasic/gods/lists/singlylinkedlist"
	"github.com/gin-gonic/gin"
)
const (
	VoteYes = iota
	VoteNo
)

var skip = 5

type VoteResp struct {
	Vote int `json:"vote"`
}
type HttpSrv struct {
	g                    *gin.Engine
	portString           string
	FollowerList         *ll.List
	HeartBeatReceived    chan bool
	UpdateTermCount      chan int
	UpdateLeaderGrpcPort chan string
	leaderGrpcPort       string
	termCount            int
	raftState   int
}
func log(i ... interface{}){
	// fmt.Print("-[HTTP]- ")
	// for x := 0; x < len(i); x++ {
		// fmt.Print(" ", i[x])
	// }
	// fmt.Println("")
	
}
func (hs *HttpSrv) Log(i interface{}){
	log(i)
}
func (hs*HttpSrv)startServer() {
	
	fmt.Println(os.Hostname())
	
	hs.Log("Listening on "+ hs.portString)
	
	hs.g.GET("/details", hs.details)
	
	p := hs.g.Group("/protocol")
	{
		p.GET("/ping", pong)
		p.GET("/heartBeat", hs.restartHeartBeatTimeout) // ?leaderPort=&termCount=
		p.GET("/voteReqFromLeader", hs.voteRequestHandler) // ?leaderPort=&termCount=
	}
	
	_ = hs.g.Run(hs.portString)
}

func pong(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{"ok":"Pong"})
}

func (hs*HttpSrv)restartHeartBeatTimeout(context *gin.Context) {
	leaderPort, termCount := context.Query("leaderPort"),context.Query("termCount")
	if termCount, err := strconv.Atoi(termCount); err == nil {
		if strings.Compare(hs.leaderGrpcPort ,leaderPort) == 0 { // from current leader
			hs.termCount = termCount
			go func() {
				hs.UpdateTermCount <- hs.termCount
				hs.UpdateLeaderGrpcPort <- leaderPort
			}()
			hs.HeartBeatReceived <- true
			context.JSON(http.StatusAccepted, nil)
			return
		}else if hs.termCount < termCount{ // from new leader
			hs.termCount = termCount
			hs.leaderGrpcPort = leaderPort
			go func() {
				hs.UpdateTermCount <- hs.termCount
				hs.UpdateLeaderGrpcPort <- leaderPort
			}()
			hs.HeartBeatReceived <- true
			context.JSON(http.StatusAccepted, nil)
			return
		}else if hs.leaderGrpcPort == ""{
			hs.termCount = termCount
			hs.leaderGrpcPort = leaderPort
			go func() {
				hs.UpdateTermCount <- hs.termCount
				hs.UpdateLeaderGrpcPort <- leaderPort
			}()
			hs.HeartBeatReceived <- true
			context.JSON(http.StatusAccepted, nil)
			return
		}
	}
	context.JSON(http.StatusExpectationFailed, nil)
}
func (hs*HttpSrv)voteRequestHandler(context *gin.Context) {
	vote := VoteResp{Vote: VoteNo}
	if hs.raftState == CANDIDATE {
		context.JSON(http.StatusOK, vote)
		return
	}
	leaderPort, termCount := context.Query("leaderPort"),context.Query("termCount")
	if termCount, err := strconv.Atoi(termCount); err == nil {
		if strings.Compare(hs.leaderGrpcPort ,leaderPort) == 0 { // from current leader
			hs.termCount = termCount
			vote.Vote = VoteYes
			go func() {
				hs.UpdateTermCount <- hs.termCount
				hs.UpdateLeaderGrpcPort <- leaderPort
			}()
		}else if hs.termCount < termCount{ // from new leader
			hs.termCount = termCount
			hs.leaderGrpcPort = leaderPort
			vote.Vote = VoteYes
			go func() {
				hs.UpdateTermCount <- hs.termCount
				hs.UpdateLeaderGrpcPort <- leaderPort
			}()
		}
	}
	context.JSON(http.StatusOK, vote)
}
func (hs *HttpSrv)loadConfig(instance string) {
	inst, _ := strconv.Atoi(instance)
	possiblePeers := ll.New()
	for i := 1; i <= 5; i++ {
		if inst != i {
			port:= os.Getenv("HTTP-"+strconv.Itoa(i))
			addr := os.Getenv("HOST" + strconv.Itoa(i)) + ":" + port
			possiblePeers.Add(addr)
		}
	}
	hs.FollowerList = possiblePeers
}
func (hs *HttpSrv)UpdatePeerList(inst string){
	hs.loadConfig(inst)
	skip --
	if skip > 0 {
		return
	}
	hs.FollowerList = pingPeers(hs.FollowerList)
	skip = 5
}
func pingPeers(peers *ll.List) *ll.List{
	var wg sync.WaitGroup
	updatedPeers := ll.New()
	// log("Updating peer list")
	for it:=peers.Iterator();it.Next(); {
		endpoint := fmt.Sprintf("%v/protocol/ping",it.Value())
		wg.Add(1)
		go func(updatedPeers *ll.List, v interface{}) {
			defer wg.Done()
			if _, pong := request(endpoint,nil); pong == 0{
				updatedPeers.Add(v)
			}
		}(updatedPeers, it.Value())
	}
	wg.Wait()
	log("Updated peers - ", updatedPeers.Size())
	return updatedPeers
}
var request =  func (url string, data interface{}) (*VoteResp,int) {
	var resp *http.Response
	var err error
	values := data
	
	log("Sending - ", url)
	
	jsonData, err := json.Marshal(values)
	if err != nil {
		fmt.Println(err)
		return nil,-1
	}
	client := http.Client{
		Timeout: 50 * time.Millisecond,
	}
	if data == nil{
		resp, err = client.Get(url)
	}else {
		resp, err = client.Post(url, "application/json",
			bytes.NewBuffer(jsonData))
	}
	if err != nil {
		return nil,-1
	}
	if strings.Contains(resp.Status, strconv.Itoa(http.StatusAccepted)) {
		return nil,0
	}
	var res VoteResp
	_ = json.NewDecoder(resp.Body).Decode(&res)
	return &res, 0
}
func (hs *HttpSrv)SendVoteReq(termCount int) int{
	voteCount := 0
	
	it := hs.FollowerList.Iterator()
	for it.Next(){
		endpoint := fmt.Sprintf(
			"%v/protocol/voteReqFromLeader?leaderPort=%s&termCount=%s",
			it.Value(),hs.portString,strconv.Itoa(termCount))
		vote, _ := request(endpoint, nil)
		if vote != nil{
			voteCount += vote.Vote
		}
	}
	return voteCount
}

func (hs *HttpSrv) SendHeartBeat(count int, portString string) {
	heartBeats := 0
	it := hs.FollowerList.Iterator()
	for it.Next() {
		if it.Value() != nil{
			endpoint := fmt.Sprintf(
				"%v/protocol/heartBeat?leaderPort=%s&termCount=%s",
				it.Value(), portString, strconv.Itoa(count))
			_, pong := request(endpoint, nil)
			if pong == 0{
				heartBeats ++
			}
		}
	}
	log("Received -", heartBeats, "heartbeat responses")
	log("State - ", hs.raftState, " term - ", hs.termCount)
}

func (hs *HttpSrv) StopHttp() {
	close(hs.UpdateLeaderGrpcPort)
	close(hs.HeartBeatReceived)
	close(hs.UpdateTermCount)
}

func (hs *HttpSrv) details(context *gin.Context) {
	host,_ := os.Hostname()
	context.JSON(http.StatusOK, gin.H{
		"Hostname": host + hs.portString,
		"RaftState": hs.raftState,
		"RaftTerm": hs.termCount,
		"leaderGrpcPort": hs.leaderGrpcPort,
		"FollowerList": hs.FollowerList,
		"portString": hs.portString,
	})
}
func StartHttp(port string, inst string) *HttpSrv{
	if strings.Compare(os.Getenv("test"), "true") != 0{
		gin.SetMode(gin.ReleaseMode)
	}
	hs := HttpSrv{
		g:                 gin.New(),
		portString:        ":"+port,
		HeartBeatReceived: make(chan bool),
		UpdateTermCount: make(chan int),
		UpdateLeaderGrpcPort: make(chan string),
		leaderGrpcPort: "",
		termCount: 0,
		raftState: FOLLOWER,
	}
	hs.loadConfig(inst)
	go hs.startServer()
	return &hs
}
