package comms

import (
	"fmt"
	
	ll "github.com/emirpasic/gods/lists/singlylinkedlist"
	"golang.org/x/net/context"
	
	comms "ishan/FSI/comms/grpc"
)

type L interface {
	UpdatePeers(list *ll.List)
	LogThis()
}

type LData struct {
	msg *comms.LogMessage
	followers *ll.List
}
func NewDataSync(source string) *LData {
	return &LData{
		msg: &comms.LogMessage{Source: source, Message: ""},
		followers: ll.New(),
	}
}
func (rd *LData) UpdatePeers(list *ll.List) {
	rd.followers = list
}
func (rd *LData) LogThis(c context.Context, lm *comms.LogMessage) (*comms.Response, error)  {
	rsp := comms.Response{Status: FAIL}
	if rd.followers.Size() == 0 {
		// I am a follower,
		// or todo -> currently leader has it empty at bootstrap
		fmt.Println("Received grpc data from leader")
		fmt.Println(lm.Source, lm.Message)
		return &comms.Response{Status: SUCCESS},nil
	}else { // I am leader, a follower sent this message
		ack := rd.followers.Size()
		l := logStruct{source: lm.Source, message: lm.Message}
		fmt.Println("Sending grpc for -", rd.followers.Size())
		for it := rd.followers.Iterator();it.Next();{ // send to my followers
			switch c := it.Value().(type) {
			case *Client:
				fmt.Println("Sending grpc to -", c)
				rsp := c.SendLogToGrpcPeer(&l)
				if rsp.Status == SUCCESS{
					ack--
				}
			}
		}
		if ack == 0 {
			rsp.Status = SUCCESS
			return &rsp, nil
		}
	}
	return nil,nil
}
