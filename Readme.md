# 一个简易的内存缓存系统

该程序需要满足一下要求:

1. 支持设定过期时间，精度为秒级
2. 支持设定最大内存，当内存超出时候做出合理的处理
3. 支持并发安全
4. 为简化编程细节，无需实现数据落地（ 不需要将数据持久化 ）。

```go
/** 
 支持过期时间和设置最⼤内存⼤小的的内存缓存库。
*/ 
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
```



### 实现内存管理 思路

- 设置一个值，用于记录Set添加变量的最大内存。当内存达到临界值时执行清理内存操作

- 通过`runtime.MemStats`读取当前软件使用内存

  ```go
  go func() {
      for  {
          time.Sleep(1 * time.Second) // 每一秒对其进行监控
          var m runtime.MemStats // 声明一个m
          runtime.ReadMemStats(&m) // 读取运行内存到m
          fmt.Printf("%d Kb\n", m.Alloc/1024) // 输出读取到的内存数据
          // 如果超出内存，则执行某某某操作( LRU、强行遍历全部内存清楚所有定时属性 )
      }
  }()
  ```
单线程思路:
  
  ```go
  /*
  	思考: 在什么时候会改变内存的大小
  	结论: 只有在进行 Set 操作的时候才会出现对内存的添加
  	所以我们需要在Set的时候进行判断:
  		1. map中是否有过期变量。
  			将过期变量删除
  		2. map中的不常用变量
  			将不常用变量清空
  	实现方法
  */
  func Set(key string, val interface{}, expire time.Duration){
   	...
  	var m runtime.MemStats // 声明一个m
  	runtime.ReadMemStats(&m) // 读取运行内存到m
      // 判断 m.Alloc 是否大于设定阈值
      ...
  }
  
  ```

  

### 实现指定时间过期

模拟Redis处理机制

- 通过设置一个 `goroutine` 设定睡眠时间，当睡眠时间结束直接删除过期元素。

  最理想方式是 将随机值控制在 `hashtable`中数组长度，每次检查直接检测当前数组的`bucket`

  但由于我们使用的是`sync.Map`包实现，而不是自定义`hashtable`。所以我们只能在Keys范围内随机一个数，并且向后检索5个元素。判断元素是否是失效的，失效的删除。

- 当使用 Get的时候判断变量是否过期，过期则删除变量并返回 nil

- 自定义 LRU处理策略 :

  - 使用双向链表存储key，当key被调用时，将其存放到头部。
  - 当内存超出时，我们删除尾部元素，直到当前内存小于最大内存限制

  


### 清理内存思路

- 当内存超出指定阈值，单独开一个线程清理过期变量。如果还是超出。则将根据算法清理不常用的变量




## 出现问题

### 1. 当删除Map中的key之后，gc只会回收value。

解决方法:  重构map。将原有map删除使用新的map进行其他操作


## 使用方法

### InitSyncMapCacheImpl 用于初始化

