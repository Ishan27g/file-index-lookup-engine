package comms

import (
	"sync"
	
	ll "github.com/emirpasic/gods/lists/singlylinkedlist"
	"golang.org/x/net/context"
	
	comms "ishan/FSI/comms/grpc"
	"ishan/FSI/parser"
)

type L interface {
	SetParser(*parser.Parser)
	UpdatePeers(list *ll.List)
	LogThis()
}

type LData struct {
	msg *comms.Query
	followers *ll.List
	pool sync.Pool
	parser         *parser.Parser
}

func (rd *LData)SetParser(p *parser.Parser){
	rd.parser = p
}


func (rd *LData) Search(ctx context.Context, query *comms.Query) (*comms.QueryResponse, error) {
	com := comms.QueryResponse{SFile: nil}
	
	if rd.followers.Size() == 0 {
		sflist := rd.parser.Find(query.Word)
		if sflist == nil{
			return &comms.QueryResponse{SFile: nil},nil
		}
		for _, sf:= range *sflist{
			com.SFile = append(com.SFile, &comms.QueryResponse_SF{
				FileName: sf.Filename,
				FileLoc:  sf.FileLoc,
				LineLoc:  sf.LineNum,
				WordLoc:  sf.WordIndex,
			})
		}
		return &com,nil
	}else { // I am leader, a follower sent this message
		lock := sync.Mutex{}
		var wg sync.WaitGroup
		for it := rd.followers.Iterator(); it.Next(); { // send to my followers
			wg.Add(1)
			go func(i interface{}, l *sync.Mutex) {
				defer wg.Done()
				switch c := i.(type) {
				case *Client:
					rsp := c.SendLookupQuery(query.Word, query.Source)
					if rsp != nil && rsp.SFile != nil {
						l.Lock()
						for _, r := range rsp.SFile {
							com.SFile = append(com.SFile, &comms.QueryResponse_SF{
								FileName:  r.FileName,
								FileLoc:   r.FileLoc,
								LineLoc:   r.LineLoc,
								WordLoc: r.WordLoc,
							})
						}
						l.Unlock()
					}
				}
			}(it.Value(), &lock)
		}
		// fmt.Println(query.Word, query.Source)
		sfList := rd.parser.Find(query.Word)
		if sfList != nil{
			lock.Lock()
			for _, sf:= range *sfList {
				com.SFile = append(com.SFile, &comms.QueryResponse_SF{
					FileName: sf.GetFileName(),
					FileLoc:  sf.FileLoc,
					LineLoc:  sf.LineNum,
					WordLoc:  sf.WordIndex,
				})
			}
			lock.Unlock()
		}
		wg.Wait()
	}
	return &com, nil
}

func NewDataSync(source string) *LData {
	return &LData{
		msg: &comms.Query{Source: source, Word: ""},
		followers: ll.New(),
		pool: sync.Pool{New: newQuery},
		parser: nil,
	}
}
func (rd *LData) UpdatePeers(list *ll.List) {
	rd.followers = list
}
func newQuery()interface{}{
	return &comms.Query{Source: "", Word: ""}
}