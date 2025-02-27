package main

import (
	"log"
	"time"
)

// SubscriptionCache 缓存服务，用于提高查询性能
type SubscriptionCache struct {
	cache          Cache
	db             *DatabaseService
	updateInterval time.Duration
	stopChan       chan struct{}
}

// NewSubscriptionCache 创建缓存服务实例
func NewSubscriptionCache(db *DatabaseService) *SubscriptionCache {
	cache := &SubscriptionCache{
		db:             db,
		updateInterval: 5 * time.Minute,
		stopChan:       make(chan struct{}),
	}

	// 初始化缓存
	if err := cache.refreshCache(); err != nil {
		log.Printf("初始化缓存失败: %v", err)
	}

	// 启动定期更新协程
	go cache.periodicUpdate()

	return cache
}

// refreshCache 刷新缓存数据，更新系统统计指标
func (sc *SubscriptionCache) refreshCache() error {
	// 获取用户总数
	userCount, err := sc.db.GetTotalUserCount()
	if err != nil {
		log.Printf("刷新缓存获取用户数失败: %v", err)
		return err
	}

	// 获取支付总额
	totalAmount, err := sc.db.GetTotalPaymentAmount()
	if err != nil {
		log.Printf("刷新缓存获取付款总额失败: %v", err)
		return err
	}

	// 获取活跃订阅数
	activeSubCount, err := sc.db.GetActiveSubscriptionsCount()
	if err != nil {
		log.Printf("刷新缓存获取活跃订阅数失败: %v", err)
		return err
	}

	// 获取本月新增订阅数
	newSubCount, err := sc.db.GetNewSubscriptionsMonth()
	if err != nil {
		log.Printf("刷新缓存获取本月新增订阅数失败: %v", err)
		return err
	}

	// 获取本月新增付费金额
	newPaymentAmount, err := sc.db.GetNewPaymentAmountMonth()
	if err != nil {
		log.Printf("刷新缓存获取本月新增付费金额失败: %v", err)
		return err
	}

	// 获取本月续订数
	renewalCount, err := sc.db.GetRenewalsMonth()
	if err != nil {
		log.Printf("刷新缓存获取本月续订数失败: %v", err)
		return err
	}

	// 获取本月续订金额
	renewalAmount, err := sc.db.GetRenewalAmountMonth()
	if err != nil {
		log.Printf("刷新缓存获取本月续订金额失败: %v", err)
		return err
	}

	// 更新缓存
	sc.cache.mutex.Lock()
	defer sc.cache.mutex.Unlock()

	sc.cache.totalUsers = userCount
	sc.cache.totalPaymentAmount = totalAmount
	sc.cache.activeSubscriptions = activeSubCount
	sc.cache.newSubscriptionsMonth = newSubCount
	sc.cache.newPaymentAmountMonth = newPaymentAmount
	sc.cache.renewalsMonth = renewalCount
	sc.cache.renewalAmountMonth = renewalAmount
	sc.cache.lastUpdated = time.Now()

	return nil
}

// periodicUpdate 定期更新缓存
func (sc *SubscriptionCache) periodicUpdate() {
	ticker := time.NewTicker(sc.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := sc.refreshCache(); err != nil {
				log.Printf("定期刷新缓存失败: %v", err)
			}
		case <-sc.stopChan:
			return
		}
	}
}

// Stop 停止缓存更新服务
func (sc *SubscriptionCache) Stop() {
	close(sc.stopChan)
}

// GetStats 获取系统统计数据
func (sc *SubscriptionCache) GetStats() SystemStats {
	sc.cache.mutex.RLock()
	defer sc.cache.mutex.RUnlock()

	return SystemStats{
		TotalUsers:            sc.cache.totalUsers,
		TotalPaymentAmount:    sc.cache.totalPaymentAmount,
		ActiveSubscriptions:   sc.cache.activeSubscriptions,
		NewSubscriptionsMonth: sc.cache.newSubscriptionsMonth,
		NewPaymentAmountMonth: sc.cache.newPaymentAmountMonth,
		RenewalsMonth:         sc.cache.renewalsMonth,
		RenewalAmountMonth:    sc.cache.renewalAmountMonth,
		LastUpdated:           sc.cache.lastUpdated,
	}
}
