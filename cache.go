package simplecachesys

import (
	"time"
)

// Cache 简单缓存系统中所需要实现的方法
type Cache interface {
	//size 是⼀一个字符串串。⽀支持以下参数: 1KB，100KB，1MB，2MB，1GB 等
	SetMaxMemory(size string) bool
	// 设置⼀一个缓存项，并且在expire时间之后过期
	Set(key string, val interface{}, expire time.Duration)
	// 获取⼀一个值
	Get(key string) (interface{}, bool)
	// 删除⼀一个值
	Del(key string) bool
	// 检测⼀一个值 是否存在
	Exists(key string) bool
	// 清空所有值
	Flush() bool
	// 返回所有的 key 的个数
	Keys() int64
}

// 单位换算
const (
	KB = 1024
	MB = KB * 1024
	GB = MB * 1024
	TB = GB * 1024
)
