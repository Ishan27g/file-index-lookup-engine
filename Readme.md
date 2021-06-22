

## Distributed file server & text search engine

A distributed file server based on the raft algorithm.
Content of the uploaded files is mapped and stored in the form of an avl tree
distributed amongst the peers, assuring efficient lookup & updates. 
The system is eventually consistent in the event of network partitions.

### Running the services
```shell
docker-compose up --build --remove-orphans
```
This will launch 5 instances of the service as independent containers.
An ingress nginx container acts as a load balancer to the services, exposing a single 
external port for client interactions.

The services are interlinked via Http + gRPC over internal docker networks. 

## Raft algorithm

- To elect a leader, the primary source for `search queries`.
- `followers` start elections if the leader fails / does not send expected heartbeats.
- `upload` & `search` queries can be forwarded to any on the peer. 
  - `upload` is processed locally by the receiving service
  - `search` is forwarded by all `followers` to the `leader`, which then concurrently
  queries the remaining `followers`.
    
### Upload '.txt' files
```shell
curl --location --request POST 'localhost:80/service/upload' --form 'files=@"./sample.txt1"' --form 'files=@"./sample2.txt1"' 
````

### Search for a word from the uploaded file(s)
```shell
curl --location --request GET 'localhost:80/service/search?word="foo"
```

### Get details (raft-state, raft-term, raft-leader, hostname, ports)
```shell
curl --location --request GET 'localhost:80/raft/details'
```

### Simulating a network partition
Once the services are up and running, a network partition can be simulated through docker.
- A call to `/raft/details` prior to network partition will list the current leader's container-name.
- To simulate a network partition with the leader, the command `docker pause $container-name` 
can be used, along with the respective name.
- A subsequent call to `/raft/details` (after the heartbeat timeout of 10s) would now list a new leader.

Further, the command `docker unpause $container-name` will reconnect to container of the old leader 
to the network. On re-joining, it will eventually become consistent with the cluster and become a follower
by virtue of receiving a higher term id.

### Stop the services
```shell
docker-compose down && docker-compose stop
```