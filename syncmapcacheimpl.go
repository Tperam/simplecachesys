package simplecachesys

import (
	"container/list"
	"fmt"
	"math/rand"
	"runtime"
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

// LRUList LRU链表，用于内存溢出后处理不常用变量
type LRUList struct {
	m sync.Mutex
	l list.List
}

// Cache接口的简单实现
type syncMapCacheImpl struct {
	// 最大内存
	memorySize uint64

	// 存放数据
	data *sync.Map

	// LRU 淘汰策略存储的map
	lruList LRUList

	// 记录key数量
	keys int64
}

// KeyUp 提升制定键的临时优先级
func (lru *LRUList) KeyUp(key interface{}) {
	lru.m.Lock()
	// 遍历链表查找我们所需要的key
	e := lru.l.Front()
	for ; e != nil; e = e.Next() {
		if e.Value == key {
			break
		}
	}
	if e == nil {
		lru.l.PushFront(key)
	} else {
		// 将当前key向前移动
		lru.l.MoveToFront(e)
	}

	lru.m.Unlock()
}

// RemoveBack 删除最后一个元素
func (lru *LRUList) RemoveBack() interface{} {
	// 上锁
	lru.m.Lock()
	// defer - 延迟解锁
	defer lru.m.Unlock()
	lastEle := lru.l.Back()
	// 判断是否为空
	if lastEle == nil {
		return nil
	}
	// 返回删除的key
	return lru.l.Remove(lastEle)
}

// Remove 删除指定元素
func (lru *LRUList) Remove(key interface{}) {
	e := lru.l.Front()
	for ; e != nil; e = e.Next() {
		if e.Value == key {
			break
		}
	}
	if e == nil {
		return
	}
	lru.l.Remove(e)
}

// InitSyncMapCacheImpl 初始化Cache实现类
func InitSyncMapCacheImpl() *syncMapCacheImpl {
	smcc := syncMapCacheImpl{
		memorySize: 500 * KB,
		data:       &sync.Map{},
	}

	go smcc.randomVerifyExpireVar()

	return &smcc
}

// SetMaxMemory 设置最大内存
// 当前自定义中 B = bytes
// eg "100KB" = 100 * 1024
func (smcc *syncMapCacheImpl) SetMaxMemory(size string) bool {

	size = strings.ToLower(strings.TrimSpace(size))

	// 切割字符串，获取单位前的数字
	numStr := size[:len(size)-2]
	// 转换成int64类型
	num, err := strconv.ParseUint(numStr, 10, 64)
	if err != nil {
		fmt.Println("输入有误,请匹配当前表达式 ^\\d+[TGMK]{0,1}B$")
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
	case "b":
		smcc.memorySize = num * B
	default:
		fmt.Println("使用了未定义的类型,请以 kb, mb, gb, tb, b结尾")
		return false
	}

	return true
}

// Set 设置一个值
func (smcc *syncMapCacheImpl) Set(key string, val interface{}, expire time.Duration) {

	// TODO 判断是否超出内存。
	flag := smcc.memoryhandle()
	if !flag {
		panic("当前值超出设定最大内存")
	}
	// 记录unix时间戳
	unix := time.Now().Add(expire).Unix()
	void := entry{
		expireTime: unix,
		val:        val,
	}
	smcc.keys++
	smcc.data.Store(key, void)
	// 单独运行线程，让其提升
	go smcc.lruList.KeyUp(key)
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
		smcc.delete(key)
		return nil, false
	}

	return e.val, ok
}

// Del 删除一个值
func (smcc *syncMapCacheImpl) Del(key string) bool {
	return smcc.delete(key)
}

// Exists 检测一个值 是否存在
func (smcc *syncMapCacheImpl) Exists(key string) bool {
	_, ok := smcc.Get(key)
	return ok
}

// Flush 清空所有值
func (smcc *syncMapCacheImpl) Flush() bool {
	// 清空 sync.map
	smcc.data = &sync.Map{}
	smcc.keys = 0
	return true
}

// Keys 返回所有的 key 的个数
func (smcc *syncMapCacheImpl) Keys() int64 {
	smcc.rangeClearExpireVar()
	return smcc.keys
}

func (smcc *syncMapCacheImpl) delete(key interface{}) bool {
	smcc.keys--
	smcc.data.Delete(key)
	smcc.lruList.Remove(key)
	return true
}

// 内存溢出处理方法
func (smcc *syncMapCacheImpl) memoryhandle() bool {
	var m runtime.MemStats   // 声明一个m
	runtime.ReadMemStats(&m) // 读取运行内存到m
	if m.Alloc > smcc.memorySize {
		// 清楚过期变量
		smcc.rangeClearExpireVar()
	}

	// 删除不常用元素，直到当前运行内存小于设定阈值
	for runtime.ReadMemStats(&m); m.Alloc > smcc.memorySize; runtime.ReadMemStats(&m) {
		if smcc.keys == 0 {
			return false
		}
		smcc.deleteLRU()
	}
	return true

}

// 遍历map清楚所有过期元素
func (smcc *syncMapCacheImpl) rangeClearExpireVar() {
	newData := sync.Map{}
	var keys int64
	// 获取当前时间
	currentTimeUnix := time.Now().Unix()
	// 判断是否有过期变量
	smcc.data.Range(func(key, val interface{}) bool {
		e := val.(entry)
		if e.expireTime > currentTimeUnix {
			keys++
			newData.Store(key, val)
		} else {
			smcc.lruList.Remove(key)
		}
		return true
	})

	smcc.data = &newData
	smcc.keys = keys
	runtime.GC()

}

// 删除最后一个变量元素
func (smcc *syncMapCacheImpl) deleteLRU() {
	key := smcc.lruList.RemoveBack()
	smcc.keys--
	smcc.data.Delete(key)
	smcc.rangeClearExpireVar()
}

// 用于定时清理缓存
func (smcc *syncMapCacheImpl) randomVerifyExpireVar() {
	for {
		// 间隔1s执行一次
		time.Sleep(1 * time.Second)
		now := time.Now().Unix()
		randNum := rand.Int63n(smcc.keys)

		var e *list.Element

		// 上锁 ，避免数据出现问题
		smcc.lruList.m.Lock()
		// 判断随机数大于还是小于 总数的一半
		if randNum > smcc.keys/2 {
			// 从后往前遍历
			e = smcc.lruList.l.Back()
			for j := smcc.keys; j >= randNum; j-- {
				e = e.Prev()
			}
		} else {
			// 从前往后遍历
			e = smcc.lruList.l.Front()
			for j := int64(0); j < randNum; j++ {
				e = e.Next()
			}
		}
		// 解锁，防止卡锁
		smcc.lruList.m.Unlock()

		// 开始对随机抽中的key进行遍历管理
		for i := randNum; i < smcc.keys && i < randNum+5 && e != nil; i++ {
			// 获取key
			key := e.Value
			// 通过 key 读取存放的entry
			v, ok := smcc.data.Load(key)
			// 判断是否存在
			if !ok {
				continue
			}
			tmp := v.(entry)
			// 验证是否过期
			if tmp.expireTime < now {
				// 删除过期变量
				smcc.delete(key)
			}
			// 继续向下执行
			e = e.Next()
		}
	}
}
