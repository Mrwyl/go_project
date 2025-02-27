package main

import (
	"log"
	"sync"
	"time"
)

// TaskScheduler 定时任务调度器
type TaskScheduler struct {
	service         *SubscriptionService
	stopChan        chan struct{}
	wg              sync.WaitGroup
	checkInterval   time.Duration // 检查即将到期订阅的时间间隔
	processInterval time.Duration // 处理已过期订阅的时间间隔
}

// NewTaskScheduler 创建新的任务调度器
func NewTaskScheduler(service *SubscriptionService) *TaskScheduler {
	return &TaskScheduler{
		service:         service,
		stopChan:        make(chan struct{}),
		checkInterval:   6 * time.Hour,  // 每6小时检查一次即将到期的订阅
		processInterval: 12 * time.Hour, // 每12小时处理一次过期的订阅
	}
}

// Start 启动所有定时任务
func (ts *TaskScheduler) Start() {
	log.Println("启动订阅系统定时任务调度器...")

	// 启动检查即将到期订阅的任务
	ts.wg.Add(1)
	go ts.runCheckExpiringTask()

	// 启动处理已过期订阅的任务
	ts.wg.Add(1)
	go ts.runProcessExpiredTask()

	log.Println("所有定时任务已启动")
}

// Stop 停止所有定时任务
func (ts *TaskScheduler) Stop() {
	log.Println("正在停止定时任务调度器...")
	close(ts.stopChan)

	// 等待所有任务完成
	done := make(chan struct{})
	go func() {
		ts.wg.Wait()
		close(done)
	}()

	// 设置超时，避免永久等待
	select {
	case <-done:
		log.Println("所有定时任务已正常停止")
	case <-time.After(10 * time.Second):
		log.Println("部分定时任务可能未能正常停止，已超时")
	}
}

// runCheckExpiringTask 运行检查即将到期订阅的定时任务
func (ts *TaskScheduler) runCheckExpiringTask() {
	defer ts.wg.Done()

	log.Printf("检查即将到期订阅任务已启动，间隔: %v", ts.checkInterval)

	// 立即执行一次
	ts.checkExpiringSubscriptions()

	// 然后按计划定时执行
	ticker := time.NewTicker(ts.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ts.checkExpiringSubscriptions()
		case <-ts.stopChan:
			log.Println("检查即将到期订阅任务收到停止信号，正在退出...")
			return
		}
	}
}

// runProcessExpiredTask 运行处理已过期订阅的定时任务
func (ts *TaskScheduler) runProcessExpiredTask() {
	defer ts.wg.Done()

	log.Printf("处理已过期订阅任务已启动，间隔: %v", ts.processInterval)

	// 立即执行一次
	ts.processExpiredSubscriptions()

	// 然后按计划定时执行
	ticker := time.NewTicker(ts.processInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ts.processExpiredSubscriptions()
		case <-ts.stopChan:
			log.Println("处理已过期订阅任务收到停止信号，正在退出...")
			return
		}
	}
}

// checkExpiringSubscriptions 执行检查即将到期订阅的逻辑
func (ts *TaskScheduler) checkExpiringSubscriptions() {
	log.Println("开始执行检查即将到期订阅任务...")
	start := time.Now()

	// 捕获可能的panic
	defer func() {
		if r := recover(); r != nil {
			log.Printf("检查即将到期订阅任务发生panic: %v", r)
		}

		log.Printf("检查即将到期订阅任务完成，耗时: %v", time.Since(start))
	}()

	// 执行业务逻辑
	ts.service.CheckExpiringSubscriptions()
}

// processExpiredSubscriptions 执行处理已过期订阅的逻辑
func (ts *TaskScheduler) processExpiredSubscriptions() {
	log.Println("开始执行处理已过期订阅任务...")
	start := time.Now()

	// 捕获可能的panic
	defer func() {
		if r := recover(); r != nil {
			log.Printf("处理已过期订阅任务发生panic: %v", r)
		}

		log.Printf("处理已过期订阅任务完成，耗时: %v", time.Since(start))
	}()

	// 执行业务逻辑
	ts.service.ProcessExpiredSubscriptions()
}
