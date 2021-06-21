package comms

type RaftService struct {
	ForwardLog chan string
	done chan bool
	raft Raft
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
		done:       make(chan bool),
		raft :      Init(instance, bootstrap),
		
	}
	go func() {
		for {
			rs.raft.Run(inst)
			rs.raft.Details()
			select {
			case <-rs.done:
				return
			case logMsg := <-rs.ForwardLog:
				rs.raft.SendLog(logMsg) // queue back in case of network partition?
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