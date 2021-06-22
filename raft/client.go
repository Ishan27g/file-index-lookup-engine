package raft

import (
	"time"
	
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	
	comms "ishan/FSI/raft/grpc"
)
func (client *Client)log(v ... interface{}) {
	// fmt.Println("[Grpc-Client]", v)
}


type Client struct {
	conn           *grpc.ClientConn
	conRaft        comms.RaftClient
	conLog         comms.LookupClient
	close          chan bool
	
}
func StartGrpcClient(grpcPeer string) *Client {
	client := Client{
		conn:           nil,
		conRaft:        nil,
		conLog:         nil,
		close: make(chan bool),
	}
	var err error
	ctx, _ := context.WithTimeout(context.Background(), 100*time.Millisecond)
	grpc.WaitForReady(false)
	client.conn, err = grpc.DialContext(ctx, grpcPeer, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		client.log("Unable to connect client - ", err)
		return nil
	}
	client.conRaft = comms.NewRaftClient(client.conn)
	client.conLog = comms.NewLookupClient(client.conn)
	
	return &client
}
func (client *Client)SendVoteReq(termCount int, leaderPort string) bool{
	votes, err := client.conRaft.RequestVotes(context.Background(), &comms.Term{
		TermCount: int32(termCount),
		LeaderPort: leaderPort,
	})
	if err != nil {
		return false
	}
	client.log(votes.Elected)
	return votes.Elected
}
func (client *Client) SendLookupQuery(word, source string) *comms.QueryResponse{ // ask peer to lookup
	msg := newQuery().(*comms.Query)
	msg.Word = word
	msg.Source = source
	data, err := client.conLog.Search(context.Background(), msg)
	if err != nil {
		return nil
	}
	return data
}
func (client *Client)Close(){
	defer client.conn.Close()
}

