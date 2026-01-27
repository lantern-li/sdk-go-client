package main

import (
	"log"
	"math"
	"math/rand"
	"sync"
)

// Distribution 定义分布接口
type Distribution interface {
	// Next 返回下一个账户的索引 (0到accountCount-1)
	Next() int
	// AccountCount 返回账户池大小
	AccountCount() int
}

// UniformDistribution 均匀分布
type UniformDistribution struct {
	accountCount int
	rng          *rand.Rand
	mu           sync.Mutex
}

// NewUniformDistribution 创建均匀分布
func NewUniformDistribution(accountCount int) *UniformDistribution {
	return &UniformDistribution{
		accountCount: accountCount,
		rng:          rand.New(rand.NewSource(rand.Int63())),
	}
}

// Next 返回均匀分布的随机账户索引
func (u *UniformDistribution) Next() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.rng.Intn(u.accountCount)
}

// AccountCount 返回账户池大小
func (u *UniformDistribution) AccountCount() int {
	return u.accountCount
}

// ZipfianDistribution 齐普夫分布
// 参考 YCSB 的实现
type ZipfianDistribution struct {
	accountCount int
	skew         float64 // 偏斜参数，值越大分布越倾斜
	alpha        float64
	zeta2theta   float64
	eta          float64
	theta        float64
	zetan        float64
	rng          *rand.Rand
	mu           sync.Mutex
}

// NewZipfianDistribution 创建齐普夫分布
// accountCount: 账户池大小
// skew: 偏斜参数，通常在0.0-1.0之间
//   - 0.0: 接近均匀分布
//   - 0.5: 中等倾斜
//   - 0.9: 高度倾斜，少数热点账户
//   - 0.99: 极度倾斜
func NewZipfianDistribution(accountCount int, skew float64) *ZipfianDistribution {
	z := &ZipfianDistribution{
		accountCount: accountCount,
		skew:         skew,
		theta:        skew,
		rng:          rand.New(rand.NewSource(rand.Int63())),
	}

	z.zeta2theta = z.zeta(2, z.theta)
	z.alpha = 1.0 / (1.0 - z.theta)
	z.zetan = z.zeta(int64(accountCount), z.theta)
	z.eta = (1 - math.Pow(2.0/float64(accountCount), 1-z.theta)) / (1 - z.zeta2theta/z.zetan)

	return z
}

// Next 返回齐普夫分布的随机账户索引
func (z *ZipfianDistribution) Next() int {
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

	ret := int(float64(z.accountCount) * math.Pow(z.eta*u-z.eta+1, z.alpha))
	return ret % z.accountCount
}

// zeta 计算 Riemann zeta 函数的近似值
func (z *ZipfianDistribution) zeta(n int64, theta float64) float64 {
	sum := 0.0
	for i := int64(0); i < n; i++ {
		sum += 1.0 / math.Pow(float64(i+1), theta)
	}
	return sum
}

// AccountCount 返回账户池大小
func (z *ZipfianDistribution) AccountCount() int {
	return z.accountCount
}

// AccountSelector 用于选择唯一的账户
type AccountSelector struct {
	dist Distribution
}

// NewAccountSelector 创建账户选择器
func NewAccountSelector(dist Distribution) *AccountSelector {
	return &AccountSelector{dist: dist}
}

// SelectUniqueAccounts 选择指定数量的唯一账户
// count: 需要选择的账户数量
// 返回账户索引数组，保证每个账户都是唯一的
func (as *AccountSelector) SelectUniqueAccounts(count int) []int {
	accounts := make([]int, 0, count)
	seen := make(map[int]bool)

	// 主策略：大幅增加尝试次数到 count*200
	maxAttempts := count * 200
	attempts := 0

	for len(accounts) < count && attempts < maxAttempts {
		account := as.dist.Next()
		if !seen[account] {
			accounts = append(accounts, account)
			seen[account] = true
		}
		attempts++
	}

	// Fallback策略：使用均匀随机而非顺序补齐
	if len(accounts) < count {
		log.Printf("Warning: Fallback triggered, got %d/%d accounts after %d attempts", len(accounts), count, attempts)
		accountCount := as.dist.AccountCount()
		rng := rand.New(rand.NewSource(rand.Int63()))

		for len(accounts) < count {
			account := rng.Intn(accountCount)
			if !seen[account] {
				accounts = append(accounts, account)
				seen[account] = true
			}
		}
	}

	return accounts
}
