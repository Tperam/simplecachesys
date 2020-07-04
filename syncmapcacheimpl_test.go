package simplecachesys

import (
	"fmt"
	"testing"
	"time"
)

func TestSyncMap(t *testing.T) {
	cache := InitSyncMapCacheImpl()

	insertData(0, 1000, cache, 5*time.Second)

	fmt.Println(cache.Get(string(1)))
	fmt.Println(cache.Get(string(10)))

	time.Sleep(6 * time.Second)

	fmt.Println("当前keys个数:", cache.Keys())
	fmt.Println("输出key为string(1)是否存在", cache.Exists(string(1)))

	insertData(0, 1000, cache, 5*time.Second)
	cache.data.Range(func(key, value interface{}) bool {
		val := (value.(entry)).val
		fmt.Println(key.(string), val)
		return true
	})
	cache.Flush()

	fmt.Println(cache.Keys())
}

func BenchmarkSetSyncMap(b *testing.B) {
	cache := InitSyncMapCacheImpl()
	insertData(0, 8192, cache, 5*time.Second)
	for i := 0; i <= 8192; i++ {
		cache.Get(string(i))
	}
}

func insertData(start, end int, cache Cache, expire time.Duration) {
	for i := start; i <= end; i++ {
		cache.Set(string(i), 10000, expire)
	}
}
