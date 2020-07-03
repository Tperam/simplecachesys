package simplecachesys

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

/*
	实现方法1:
		通过sync.map进行实现
		内存超出执行策略为:
			1. 清空所有过期数据
			2. lru -> allkeys-lru 当内存不足以容纳新写入数据时，在键空间中，移除最近最少使用的key
*/

// 定义一个void结构体
type entry struct {
	expireTime int64       // 过期时间 unix
	val        interface{} // 存入值
}

// Cache接口的简单实现
type syncMapCacheImpl struct {
	// 最大内存
	memorySize uint64

	// 存放数据
	data sync.Map

	// 记录key数量
	keys int64
}

// InitSyncMapCacheImpl 初始化Cache实现类
func InitSyncMapCacheImpl() syncMapCacheImpl {

	return syncMapCacheImpl{
		memorySize: 1 * MB,
		data:       sync.Map{},
	}
}

// SetMaxMemory 设置最大内存
// 当前自定义中 b = bytes 不使用Bit
func (smcc *syncMapCacheImpl) SetMaxMemory(size string) bool {

	size = strings.ToLower(strings.TrimSpace(size))

	// 切割字符串，获取单位前的数字
	numStr := size[:len(size)-2]
	// 转换成int64类型
	num, err := strconv.ParseUint(numStr, 10, 64)
	if err != nil {
		return false
	}
	// 切割单位
	unit := size[len(size)-2:]

	switch unit {
	case "tb":
		smcc.memorySize = num * TB
	case "gb":
		smcc.memorySize = num * GB
	case "mb":
		smcc.memorySize = num * MB
	case "kb":
		smcc.memorySize = num * KB
	default:
		fmt.Println("使用了未定义的类型,请以 kb, mb, gb, tb 结尾")
		return false
	}

	return true
}

// Set 设置一个值
func (smcc *syncMapCacheImpl) Set(key string, val interface{}, expire time.Duration) {

	// TODO 判断是否超出内存。

	// 记录unix时间戳
	unix := time.Now().Add(expire).Unix()
	void := entry{
		expireTime: unix,
		val:        val,
	}
	smcc.keys++
	smcc.data.Store(key, void)

}

// Get 获取一个值
func (smcc *syncMapCacheImpl) Get(key string) (interface{}, bool) {
	v, ok := smcc.data.Load(key)
	// 判断是否存在数据
	if !ok {
		return nil, ok
	}
	e := v.(entry)
	// 判断是否是过期数据
	if e.expireTime < time.Now().Unix() {
		smcc.data.Delete(key)
		return nil, false
	}

	return e.val, ok
}

// Del 删除一个值
func (smcc *syncMapCacheImpl) Del(key string) bool {
	smcc.keys--
	smcc.data.Delete(key)
	return true
}

// Exists 检测一个值 是否存在
func (smcc *syncMapCacheImpl) Exists(key string) bool {
	_, ok := smcc.Get(key)
	return ok
}

// Flush 清空所有值
func (smcc *syncMapCacheImpl) Flush() bool {
	// 清空 sync.map
	smcc.data = sync.Map{}
	smcc.keys = 0
	return true
}

// Keys 返回所有的 key 的个数
func (smcc *syncMapCacheImpl) Keys() int64 {
	smcc.rangeClearExpireVar()
	return smcc.keys
}

// 遍历map清楚所有过期元素
func (smcc *syncMapCacheImpl) rangeClearExpireVar() {
	// 获取当前时间
	currentTimeUnix := time.Now().Unix()
	// 判断是否有过期变量
	smcc.data.Range(func(key, val interface{}) bool {
		e := val.(entry)
		if e.expireTime > currentTimeUnix {
			return true
		}
		// 删除过期变量
		smcc.Del(key.(string))
		return true
	})

}
