package main

import (
	"log"
	"math"
	"math/rand"
	"sync"
)

// Distribution 定义分布接口
type Distribution interface {
	// Next 返回下一个key的索引 (0到recordCount-1)
	Next() int64
	// RecordCount 返回键空间大小
	RecordCount() int64
}

// UniformDistribution 均匀分布
type UniformDistribution struct {
	recordCount int64
	rng         *rand.Rand
	mu          sync.Mutex
}

// NewUniformDistribution 创建均匀分布
func NewUniformDistribution(recordCount int64) *UniformDistribution {
	return &UniformDistribution{
		recordCount: recordCount,
		rng:         rand.New(rand.NewSource(rand.Int63())),
	}
}

// Next 返回均匀分布的随机key索引
func (u *UniformDistribution) Next() int64 {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.rng.Int63n(u.recordCount)
}

// RecordCount 返回键空间大小
func (u *UniformDistribution) RecordCount() int64 {
	return u.recordCount
}

// ZipfianDistribution 齐普夫分布
// 参考 YCSB 的实现
type ZipfianDistribution struct {
	recordCount int64
	skew        float64 // 偏斜参数，值越大分布越倾斜
	alpha       float64
	zeta2theta  float64
	eta         float64
	theta       float64
	zetan       float64
	rng         *rand.Rand
	mu          sync.Mutex
}

// NewZipfianDistribution 创建齐普夫分布
// recordCount: 键空间大小
// skew: 偏斜参数，通常在0.0-2.0之间
//   - 0.0: 接近均匀分布
//   - 0.99: YCSB默认值，中等倾斜
//   - 1.0以上: 高度倾斜，少数热点key
func NewZipfianDistribution(recordCount int64, skew float64) *ZipfianDistribution {
	z := &ZipfianDistribution{
		recordCount: recordCount,
		skew:        skew,
		theta:       skew,
		rng:         rand.New(rand.NewSource(rand.Int63())),
	}

	z.zeta2theta = z.zeta(2, z.theta)
	z.alpha = 1.0 / (1.0 - z.theta)
	z.zetan = z.zeta(recordCount, z.theta)
	z.eta = (1 - math.Pow(2.0/float64(recordCount), 1-z.theta)) / (1 - z.zeta2theta/z.zetan)

	return z
}

// Next 返回齐普夫分布的随机key索引
func (z *ZipfianDistribution) Next() int64 {
	z.mu.Lock()
	defer z.mu.Unlock()

	u := z.rng.Float64()
	uz := u * z.zetan

	if uz < 1.0 {
		return 0
	}

	if uz < 1.0+math.Pow(0.5, z.theta) {
		return 1
	}

	ret := int64(float64(z.recordCount) * math.Pow(z.eta*u-z.eta+1, z.alpha))
	return ret % z.recordCount
}

// zeta 计算 Riemann zeta 函数的近似值
func (z *ZipfianDistribution) zeta(n int64, theta float64) float64 {
	sum := 0.0
	for i := int64(0); i < n; i++ {
		sum += 1.0 / math.Pow(float64(i+1), theta)
	}
	return sum
}

// RecordCount 返回键空间大小
func (z *ZipfianDistribution) RecordCount() int64 {
	return z.recordCount
}

// KeySelector 用于选择唯一的key
type KeySelector struct {
	dist Distribution
}

// NewKeySelector 创建key选择器
func NewKeySelector(dist Distribution) *KeySelector {
	return &KeySelector{dist: dist}
}

// SelectUniqueKeys 选择指定数量的唯一key
// count: 需要选择的key数量
// 返回key索引数组，保证每个key都是唯一的
func (ks *KeySelector) SelectUniqueKeys(count int) []int64 {
	keys := make([]int64, 0, count)
	seen := make(map[int64]bool)

	// 主策略：大幅增加尝试次数到 count*150
	maxAttempts := count * 150
	attempts := 0

	for len(keys) < count && attempts < maxAttempts {
		key := ks.dist.Next()
		if !seen[key] {
			keys = append(keys, key)
			seen[key] = true
		}
		attempts++
	}

	// Fallback策略：使用均匀随机而非顺序补齐
	if len(keys) < count {
		log.Printf("Warning: Fallback triggered, got %d/%d keys after %d attempts", len(keys), count, attempts)
		recordCount := ks.dist.RecordCount()
		rng := rand.New(rand.NewSource(rand.Int63()))

		for len(keys) < count {
			key := rng.Int63n(recordCount)
			if !seen[key] {
				keys = append(keys, key)
				seen[key] = true
			}
		}
	}

	return keys
}
