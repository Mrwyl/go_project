package main

import (
	"fmt"
	"log"
	"time"
)

// NotificationService 处理系统通知
type NotificationService struct {
	db *DatabaseService
}

// NewNotificationService 创建通知服务实例
func NewNotificationService(db *DatabaseService) *NotificationService {
	return &NotificationService{db: db}
}

// SendExpirationNotice 发送即将到期通知
func (s *NotificationService) SendExpirationNotice(userID, subscriptionID int64) error {
	// 记录日志
	log.Printf("正在发送订阅到期通知: 用户ID=%d, 订阅ID=%d", userID, subscriptionID)

	// 获取用户信息
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		log.Printf("获取用户信息失败: %v", err)
		return fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 获取订阅信息
	subscription, err := s.db.GetSubscriptionByID(subscriptionID)
	if err != nil {
		log.Printf("获取订阅信息失败: %v", err)
		return fmt.Errorf("获取订阅信息失败: %w", err)
	}

	// 构建通知内容
	content := fmt.Sprintf(
		"亲爱的%s，您的订阅将于%s到期，请考虑是否续订。",
		user.Name,
		subscription.EndDate.Format("2006-01-02"),
	)

	// 在实际系统中，这里会发送邮件或推送通知
	// 这里仅记录日志和存储通知记录
	log.Printf("向用户 %d 发送订阅到期通知: %s", userID, content)

	// 记录通知
	notification := &Notification{
		UserID:         userID,
		SubscriptionID: subscriptionID,
		Type:           "expiration_notice",
		Content:        content,
		SentAt:         time.Now(),
		Status:         "sent",
	}

	err = s.saveNotification(notification)
	if err != nil {
		log.Printf("保存通知记录失败: %v", err)
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	return nil
}

// SendRenewalConfirmation 发送续约成功通知
func (s *NotificationService) SendRenewalConfirmation(userID, subscriptionID int64) error {
	// 记录日志
	log.Printf("正在发送续约确认通知: 用户ID=%d, 订阅ID=%d", userID, subscriptionID)

	// 获取用户信息
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		log.Printf("获取用户信息失败: %v", err)
		return fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 获取订阅信息
	subscription, err := s.db.GetSubscriptionByID(subscriptionID)
	if err != nil {
		log.Printf("获取订阅信息失败: %v", err)
		return fmt.Errorf("获取订阅信息失败: %w", err)
	}

	// 构建通知内容
	content := fmt.Sprintf(
		"亲爱的%s，您的订阅已成功续约，下一个周期将于%s开始。",
		user.Name,
		subscription.EndDate.Format("2006-01-02"),
	)

	// 在实际系统中，这里会发送邮件或推送通知
	log.Printf("向用户 %d 发送续约成功通知: %s", userID, content)

	// 记录通知
	notification := &Notification{
		UserID:         userID,
		SubscriptionID: subscriptionID,
		Type:           "renewal_confirmation",
		Content:        content,
		SentAt:         time.Now(),
		Status:         "sent",
	}

	err = s.saveNotification(notification)
	if err != nil {
		log.Printf("保存通知记录失败: %v", err)
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	return nil
}

// SendCancelConfirmation 发送取消续约确认通知
func (s *NotificationService) SendCancelConfirmation(userID, subscriptionID int64) error {
	// 记录日志
	log.Printf("正在发送取消续约通知: 用户ID=%d, 订阅ID=%d", userID, subscriptionID)

	// 获取用户信息
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		log.Printf("获取用户信息失败: %v", err)
		return fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 获取订阅信息
	subscription, err := s.db.GetSubscriptionByID(subscriptionID)
	if err != nil {
		log.Printf("获取订阅信息失败: %v", err)
		return fmt.Errorf("获取订阅信息失败: %w", err)
	}

	// 构建通知内容
	content := fmt.Sprintf(
		"亲爱的%s，我们已确认您的取消续约请求，您的订阅服务将持续到%s。",
		user.Name,
		subscription.EndDate.Format("2006-01-02"),
	)

	// 在实际系统中，这里会发送邮件或推送通知
	log.Printf("向用户 %d 发送取消续约确认通知: %s", userID, content)

	// 记录通知
	notification := &Notification{
		UserID:         userID,
		SubscriptionID: subscriptionID,
		Type:           "cancel_confirmation",
		Content:        content,
		SentAt:         time.Now(),
		Status:         "sent",
	}

	err = s.saveNotification(notification)
	if err != nil {
		log.Printf("保存通知记录失败: %v", err)
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	return nil
}

// SendSubscriptionEndedNotice 发送订阅结束通知
func (s *NotificationService) SendSubscriptionEndedNotice(userID, subscriptionID int64) error {
	// 记录日志
	log.Printf("正在发送订阅结束通知: 用户ID=%d, 订阅ID=%d", userID, subscriptionID)

	// 获取用户信息
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		log.Printf("获取用户信息失败: %v", err)
		return fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 构建通知内容
	content := fmt.Sprintf(
		"亲爱的%s，您的订阅已结束，如需继续使用服务，请重新订阅。",
		user.Name,
	)

	// 在实际系统中，这里会发送邮件或推送通知
	log.Printf("向用户 %d 发送订阅结束通知: %s", userID, content)

	// 记录通知
	notification := &Notification{
		UserID:         userID,
		SubscriptionID: subscriptionID,
		Type:           "subscription_ended",
		Content:        content,
		SentAt:         time.Now(),
		Status:         "sent",
	}

	err = s.saveNotification(notification)
	if err != nil {
		log.Printf("保存通知记录失败: %v", err)
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	return nil
}

// saveNotification 保存通知记录到数据库
func (s *NotificationService) saveNotification(notification *Notification) error {
	query := `INSERT INTO notifications 
              (user_id, subscription_id, type, content, sent_at, status) 
              VALUES (?, ?, ?, ?, ?, ?)`

	_, err := s.db.db.Exec(
		query,
		notification.UserID,
		notification.SubscriptionID,
		notification.Type,
		notification.Content,
		notification.SentAt,
		notification.Status,
	)

	if err != nil {
		return fmt.Errorf("插入通知记录失败: %w", err)
	}

	return nil
}
