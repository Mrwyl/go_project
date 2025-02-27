package main

import (
	"errors"
	"fmt"
	"log"
	"time"
)

const (
	// 订阅价格（简化起见，统一价格）
	SubscriptionPrice = 29.99
)

// SubscriptionService 提供订阅系统业务逻辑
type SubscriptionService struct {
	db              *DatabaseService
	cache           *SubscriptionCache
	notificationSvc *NotificationService
}

// NewSubscriptionService 创建订阅服务实例
func NewSubscriptionService(dsn string) (*SubscriptionService, error) {
	db, err := NewDatabaseService(dsn)
	if err != nil {
		log.Printf("创建数据库服务失败: %v", err)
		return nil, fmt.Errorf("创建数据库服务失败: %w", err)
	}

	cache := NewSubscriptionCache(db)
	notificationSvc := NewNotificationService(db)

	svc := &SubscriptionService{
		db:              db,
		cache:           cache,
		notificationSvc: notificationSvc,
	}

	return svc, nil
}

// 用户API - 获取订阅信息
func (s *SubscriptionService) GetUserSubscriptionInfo(userID int64) ([]Subscription, error) {
	log.Printf("获取用户 %d 的订阅信息", userID)
	return s.db.GetUserSubscriptions(userID)
}

// 用户API - 获取付款记录
func (s *SubscriptionService) GetUserPaymentHistory(userID int64) ([]Payment, error) {
	log.Printf("获取用户 %d 的支付记录", userID)
	return s.db.GetUserPayments(userID)
}

// 管理API - 获取实时统计数据
func (s *SubscriptionService) GetSystemStats() SystemStats {
	log.Printf("获取系统统计数据")
	return s.cache.GetStats()
}

// 管理API - 按时间段查询付费数据
func (s *SubscriptionService) GetPaymentStatsByTimeRange(query TimeRangeQuery) (*TimeRangeStats, error) {
	log.Printf("按时间段查询付费数据: %s - %s",
		query.StartTime.Format("2006-01-02"),
		query.EndTime.Format("2006-01-02"))

	return s.db.GetPaymentStatsByTimeRange(query.StartTime, query.EndTime)
}

// 创建新用户
func (s *SubscriptionService) CreateUser(name, email string) (int64, error) {
	if name == "" || email == "" {
		return 0, errors.New("用户名和邮箱不能为空")
	}

	log.Printf("创建新用户: name=%s, email=%s", name, email)

	user := &User{
		Name:  name,
		Email: email,
	}

	userID, err := s.db.CreateUser(user)
	if err != nil {
		log.Printf("创建用户失败: %v", err)
		return 0, err
	}

	// 为用户创建未激活订阅
	err = s.CreateInactiveSubscription(userID)
	if err != nil {
		log.Printf("为用户 %d 创建初始未激活订阅失败: %v", userID, err)
		return userID, fmt.Errorf("创建用户成功但初始化订阅失败: %w", err)
	}

	log.Printf("用户创建成功，ID: %d", userID)
	return userID, nil
}

// 创建未激活订阅
func (s *SubscriptionService) CreateInactiveSubscription(userID int64) error {
	log.Printf("为用户 %d 创建未激活订阅", userID)

	now := time.Now()

	// 未激活订阅默认不设置结束日期
	subscription := &Subscription{
		UserID:            userID,
		Plan:              "basic",
		StartDate:         now,
		EndDate:           now, // 未激活状态下结束日期与开始日期相同
		Status:            StatusInactive,
		NotificationSent:  false,
		RenewalPreference: "undecided",
	}

	// 开始事务
	tx, err := s.db.BeginTx()
	if err != nil {
		log.Printf("开始事务失败: %v", err)
		return fmt.Errorf("开始事务失败: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
			log.Printf("事务回滚")
		}
	}()

	// 创建订阅记录
	result, err := tx.Exec(
		`INSERT INTO subscriptions 
        (user_id, plan, start_date, end_date, status, notification_sent, renewal_preference) 
        VALUES (?, ?, ?, ?, ?, ?, ?)`,
		subscription.UserID,
		subscription.Plan,
		subscription.StartDate,
		subscription.EndDate,
		subscription.Status,
		subscription.NotificationSent,
		subscription.RenewalPreference,
	)

	if err != nil {
		log.Printf("创建未激活订阅失败: %v", err)
		return fmt.Errorf("创建未激活订阅失败: %w", err)
	}

	// 获取插入的订阅ID
	subID, err := result.LastInsertId()
	if err != nil {
		log.Printf("获取订阅ID失败: %v", err)
		return fmt.Errorf("获取订阅ID失败: %w", err)
	}

	log.Printf("未激活订阅创建成功，ID: %d", subID)

	// 提交事务
	if err = tx.Commit(); err != nil {
		log.Printf("提交事务失败: %v", err)
		return fmt.Errorf("提交事务失败: %w", err)
	}

	// 刷新缓存
	if err = s.cache.refreshCache(); err != nil {
		log.Printf("刷新缓存失败: %v", err)
	}

	return nil
}

// 激活订阅（支付首次订阅费）
func (s *SubscriptionService) ActivateSubscription(userID int64, plan string) error {
	log.Printf("激活用户 %d 的订阅，计划: %s", userID, plan)

	// 检查是否有未激活订阅
	subscriptions, err := s.db.GetUserSubscriptions(userID)
	if err != nil {
		log.Printf("获取用户订阅失败: %v", err)
		return err
	}

	var inactiveSubscription *Subscription
	for _, sub := range subscriptions {
		if sub.Status == StatusInactive {
			inactiveSubscription = &sub
			break
		}
	}

	if inactiveSubscription == nil {
		log.Printf("找不到未激活的订阅")
		return errors.New("找不到未激活的订阅")
	}

	// 开始事务
	tx, err := s.db.BeginTx()
	if err != nil {
		log.Printf("开始事务失败: %v", err)
		return fmt.Errorf("开始事务失败: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
			log.Printf("事务回滚")
		}
	}()

	// 更新订阅信息
	now := time.Now()
	endDate := now.AddDate(0, 1, 0) // 订阅一个月

	_, err = tx.Exec(
		`UPDATE subscriptions 
        SET plan = ?, status = ?, start_date = ?, end_date = ?, notification_sent = ? 
        WHERE id = ?`,
		plan,
		StatusSubscribed,
		now,
		endDate,
		false, // 重置通知状态
		inactiveSubscription.ID,
	)

	if err != nil {
		log.Printf("更新订阅状态失败: %v", err)
		return fmt.Errorf("更新订阅状态失败: %w", err)
	}

	// 创建支付记录
	_, err = tx.Exec(
		`INSERT INTO payments 
        (user_id, subscription_id, amount, payment_date, status, type) 
        VALUES (?, ?, ?, ?, ?, ?)`,
		userID,
		inactiveSubscription.ID,
		SubscriptionPrice,
		now,
		"success",
		"initial",
	)

	if err != nil {
		log.Printf("创建支付记录失败: %v", err)
		return fmt.Errorf("创建支付记录失败: %w", err)
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		log.Printf("提交事务失败: %v", err)
		return fmt.Errorf("提交事务失败: %w", err)
	}

	log.Printf("用户 %d 的订阅激活成功", userID)

	// 刷新缓存
	if err = s.cache.refreshCache(); err != nil {
		log.Printf("刷新缓存失败: %v", err)
	}

	return nil
}

// 处理续订请求
func (s *SubscriptionService) RenewSubscription(request RenewalRequest) error {
	log.Printf("处理续订请求: 订阅ID=%d, 用户ID=%d", request.SubscriptionID, request.UserID)

	// 获取订阅信息
	subscription, err := s.db.GetSubscriptionByID(request.SubscriptionID)
	if err != nil {
		log.Printf("获取订阅信息失败: %v", err)
		return err
	}

	// 验证用户ID
	if subscription.UserID != request.UserID {
		log.Printf("用户ID不匹配: 订阅所属用户=%d, 请求用户=%d", subscription.UserID, request.UserID)
		return errors.New("用户ID与订阅不匹配")
	}

	// 验证订阅状态
	if subscription.Status != StatusSubscribed {
		log.Printf("订阅状态不适合续订: %s", subscription.Status)
		return errors.New("只有已订阅状态的订阅可以续约")
	}

	// 开始事务
	tx, err := s.db.BeginTx()
	if err != nil {
		log.Printf("开始事务失败: %v", err)
		return fmt.Errorf("开始事务失败: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
			log.Printf("事务回滚")
		}
	}()

	// 计算新的结束日期
	newEndDate := subscription.EndDate.AddDate(0, 1, 0)

	// 更新订阅状态和结束日期
	_, err = tx.Exec(
		`UPDATE subscriptions 
    SET status = ?, renewal_preference = ?, end_date = ? 
    WHERE id = ?`,
		StatusRenewed,
		"yes",
		newEndDate,
		subscription.ID,
	)

	if err != nil {
		log.Printf("更新订阅状态失败: %v", err)
		return fmt.Errorf("更新订阅状态失败: %w", err)
	}

	// 创建支付记录
	now := time.Now()
	_, err = tx.Exec(
		`INSERT INTO payments 
        (user_id, subscription_id, amount, payment_date, status, type) 
        VALUES (?, ?, ?, ?, ?, ?)`,
		request.UserID,
		request.SubscriptionID,
		request.Amount,
		now,
		"success",
		"renewal",
	)

	if err != nil {
		log.Printf("创建续订支付记录失败: %v", err)
		return fmt.Errorf("创建续订支付记录失败: %w", err)
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		log.Printf("提交事务失败: %v", err)
		return fmt.Errorf("提交事务失败: %w", err)
	}

	log.Printf("订阅 %d 续约成功", subscription.ID)

	// 发送续约成功通知
	go func() {
		if err := s.notificationSvc.SendRenewalConfirmation(subscription.UserID, subscription.ID); err != nil {
			log.Printf("发送续约确认通知失败: %v", err)
		}
	}()

	// 刷新缓存
	if err = s.cache.refreshCache(); err != nil {
		log.Printf("刷新缓存失败: %v", err)
	}

	return nil
}

// 取消续订
func (s *SubscriptionService) CancelRenewal(request CancelRenewalRequest) error {
	log.Printf("处理取消续订请求: 订阅ID=%d, 用户ID=%d", request.SubscriptionID, request.UserID)

	// 获取订阅信息
	subscription, err := s.db.GetSubscriptionByID(request.SubscriptionID)
	if err != nil {
		log.Printf("获取订阅信息失败: %v", err)
		return err
	}

	// 验证用户ID
	if subscription.UserID != request.UserID {
		log.Printf("用户ID不匹配: 订阅所属用户=%d, 请求用户=%d", subscription.UserID, request.UserID)
		return errors.New("用户ID与订阅不匹配")
	}

	// 验证订阅状态
	if subscription.Status != StatusSubscribed && subscription.Status != StatusRenewed {
		log.Printf("订阅状态不适合取消续约: %s", subscription.Status)
		return errors.New("只有已订阅或已续约的订阅可以取消续约")
	}

	// 更新订阅状态为已退订
	err = s.db.UpdateSubscriptionStatus(subscription.ID, StatusUnsubscribed)
	if err != nil {
		log.Printf("更新订阅状态失败: %v", err)
		return err
	}

	// 更新续订偏好
	err = s.db.UpdateRenewalPreference(subscription.ID, "no")
	if err != nil {
		log.Printf("更新续订偏好失败: %v", err)
		return err
	}

	log.Printf("订阅 %d 已标记为已退订", subscription.ID)

	// 发送取消续约通知
	go func() {
		if err := s.notificationSvc.SendCancelConfirmation(subscription.UserID, subscription.ID); err != nil {
			log.Printf("发送取消续约确认通知失败: %v", err)
		}
	}()

	// 刷新缓存
	if err = s.cache.refreshCache(); err != nil {
		log.Printf("刷新缓存失败: %v", err)
	}

	return nil
}

// 检查即将到期的订阅并发送通知
func (s *SubscriptionService) CheckExpiringSubscriptions() {
	log.Printf("开始检查即将到期的订阅")

	subscriptions, err := s.db.GetExpiringSubscriptionsForNotification()
	if err != nil {
		log.Printf("获取即将到期订阅失败: %v", err)
		return
	}

	log.Printf("找到 %d 个需要发送通知的即将到期订阅", len(subscriptions))

	for _, sub := range subscriptions {
		// 发送即将到期通知
		err = s.notificationSvc.SendExpirationNotice(sub.UserID, sub.ID)
		if err != nil {
			log.Printf("发送订阅 %d 到期通知失败: %v", sub.ID, err)
			continue
		}

		// 更新通知已发送标志
		err = s.db.UpdateSubscriptionNotificationSent(sub.ID, true)
		if err != nil {
			log.Printf("更新订阅 %d 通知状态失败: %v", sub.ID, err)
		} else {
			log.Printf("订阅 %d 到期通知已发送", sub.ID)
		}
	}
}

// 处理已过期订阅
func (s *SubscriptionService) ProcessExpiredSubscriptions() {
	log.Printf("开始处理已过期的订阅")

	subscriptions, err := s.db.GetExpiredSubscriptions()
	if err != nil {
		log.Printf("获取已过期订阅失败: %v", err)
		return
	}

	log.Printf("找到 %d 个已过期的订阅需要处理", len(subscriptions))

	for _, sub := range subscriptions {
		var newStatus string

		// 根据当前状态判断转换为什么状态
		switch sub.Status {
		case StatusRenewed:
			// 已续约 -> 已订阅（开始新周期）
			newStatus = StatusSubscribed

			// // 更新订阅日期为下一个周期
			// err = s.db.UpdateSubscriptionDates(sub.ID, sub.EndDate, sub.EndDate.AddDate(0, 1, 0))
			// if err != nil {
			// 	log.Printf("更新订阅 %d 日期失败: %v", sub.ID, err)
			// 	continue
			// }

			// 重置通知状态
			err = s.db.UpdateSubscriptionNotificationSent(sub.ID, false)
			if err != nil {
				log.Printf("重置订阅 %d 通知状态失败: %v", sub.ID, err)
			}

			log.Printf("订阅 %d 状态从已续约更新为已订阅，进入新周期", sub.ID)
			// 重置续订偏好为undecided
			err = s.db.UpdateRenewalPreference(sub.ID, "undecided")
			if err != nil {
				log.Printf("重置订阅 %d 续订偏好失败: %v", sub.ID, err)
			}

			log.Printf("订阅 %d 状态从已续约更新为已订阅，进入新周期", sub.ID)

		case StatusUnsubscribed, StatusSubscribed:
			// 已退订/已订阅但没有操作 -> 未激活
			newStatus = StatusInactive

			// 发送订阅结束通知
			go func(userID, subscriptionID int64) {
				if err := s.notificationSvc.SendSubscriptionEndedNotice(userID, subscriptionID); err != nil {
					log.Printf("发送订阅结束通知失败: %v", err)
				}
			}(sub.UserID, sub.ID)

			log.Printf("订阅 %d 状态更新为未激活", sub.ID)
		}

		// 更新状态
		err = s.db.UpdateSubscriptionStatus(sub.ID, newStatus)
		if err != nil {
			log.Printf("更新订阅 %d 状态为 %s 失败: %v", sub.ID, newStatus, err)
			continue
		}
	}

	// 刷新缓存
	if err = s.cache.refreshCache(); err != nil {
		log.Printf("刷新缓存失败: %v", err)
	}
}

// 关闭服务
func (s *SubscriptionService) Close() error {
	// 停止缓存更新
	s.cache.Stop()

	// 关闭数据库连接
	if err := s.db.Close(); err != nil {
		log.Printf("关闭数据库连接失败: %v", err)
		return fmt.Errorf("关闭数据库连接失败: %w", err)
	}

	log.Printf("订阅服务已关闭")
	return nil
}
