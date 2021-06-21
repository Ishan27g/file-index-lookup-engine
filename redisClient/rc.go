package redisClient

import (
	"fmt"
	
	"github.com/go-redis/redis/v8"
	"golang.org/x/net/context"
)
var ctx = context.Background()
type Rdb interface {
	Add(string, string)
	Get(string)[]string
}
type Rclient struct {
	client *redis.Client
	source string
}
func (rc * Rclient)Get(key string)[]string {
	value, err := rc.client.MGet(ctx, key).Result()
	if err != nil {
		fmt.Println(err)
	}
	var ss []string
	for _, v:= range value{
		switch s := v.(type) {
		case string:
			ss = append(ss, s)
		}
	}
	return ss
}
func (rc * Rclient)Add(key string, i string){
	err := rc.client.Set(ctx, key, i, 0).Err()
	if err != nil {
		fmt.Println(err)
	}
}
func Init(source string) Rdb{
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		Password: "",
		DB: 0,
	})
	fmt.Println(client.Ping(ctx))
	return &Rclient{
		client: client,
		source: source,
	}
}
