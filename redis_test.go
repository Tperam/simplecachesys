package simplecachesys

import (
	"fmt"
	"testing"

	"github.com/garyburd/redigo/redis"
)

func BenchmarkSetRedisStr(b *testing.B) {
	c, err := redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		fmt.Println("connect to redis error", err)
	}

	insertDataToRedis(c, 0, 8192)

	for i := 0; i <= 8192; i++ {
		c.Do("GET", string(i))
	}

	c.Close()
}

func insertDataToRedis(c redis.Conn, start, end int) {
	for i := start; i <= end; i++ {
		_, err := c.Do("SET", string(i), 1000, "EX", "5")
		if err != nil {
			fmt.Println("redis set failed:", err)
		}
	}
}
