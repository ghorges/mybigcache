package bigcache

import (
	"time"
)

type onRemoveCallback func(wrappedEntry []byte, reason string)

type BigCache struct {
	shards      []*cacheShard
	shardSize   int
	lifeWindow  time.Duration
	cleanWindow time.Duration
	shardMask   uint64
	hash        fnv64a
	OnRemove    onRemoveCallback
}

func NewBigCache(shardSize int, lifeWindow time.Duration, cleanWindow time.Duration, onRemove onRemoveCallback, initSize int) *BigCache {
	return newBigCache(shardSize, lifeWindow, cleanWindow, onRemove, initSize)
}

func DefaultBigCache() *BigCache {
	return newBigCache(1024, 10*time.Minute, 5*time.Minute, removeDefaultCall, 100)
}

func newBigCache(shardSize int, lifeWindow time.Duration, cleanWindow time.Duration, onRemove onRemoveCallback, initSize int) *BigCache {
	if isPowerOfTwo(shardSize) == false {
		return nil
	}

	bigCache := &BigCache{
		shardSize:   shardSize,
		lifeWindow:  lifeWindow,
		cleanWindow: cleanWindow,
		shardMask:   uint64(shardSize) - 1,
		OnRemove:    onRemove,
	}

	bigCache.shards = make([]*cacheShard, shardSize)

	for i := 0; i < shardSize; i++ {
		bigCache.shards[i] = initShard(initSize, int64(lifeWindow.Seconds()), onRemove)
	}

	if bigCache.lifeWindow > 0 {
		go func() {
			ticker := time.NewTicker(bigCache.cleanWindow)
			defer ticker.Stop()
			for {
				select {
				case t := <-ticker.C:
					bigCache.cleanUp(int64(t.Unix()))
				}
			}
		}()
	}
	return bigCache
}

func isPowerOfTwo(number int) bool {
	return (number & (number - 1)) == 0
}

func (cache *BigCache) Get(key string) ([]byte, error) {
	hashKey := cache.hash.Sum64(key)
	shard := cache.getShard(hashKey)
	return shard.get(hashKey, key)
}

func (cache *BigCache) Set(key string, data []byte) error {
	hashKey := cache.hash.Sum64(key)
	shard := cache.getShard(hashKey)
	return shard.set(key, hashKey, data)
}

func (cache *BigCache) Del(key string) error {
	hashKey := cache.hash.Sum64(key)
	shard := cache.getShard(hashKey)
	return shard.del(hashKey, key)
}

func (cache *BigCache) cleanUp(currentTimestamp int64) {
	for _, shard := range cache.shards {
		shard.cleanUp(currentTimestamp)
	}
}

func (cache *BigCache) getShard(hashKey uint64) *cacheShard {
	return cache.shards[hashKey&cache.shardMask]
}

func removeDefaultCall(wrappedEntry []byte, reason string) {
}
