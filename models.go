package main

import (
	"sync"
	"time"
)

// 订阅状态常量
const (
	StatusInactive     = "inactive"     // 未激活
	StatusSubscribed   = "subscribed"   // 已订阅
	StatusRenewed      = "renewed"      // 已续约
	StatusUnsubscribed = "unsubscribed" // 已退订
)

// 模型定义
type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Subscription struct {
	ID                int64     `json:"id"`
	UserID            int64     `json:"user_id"`
	Plan              string    `json:"plan"`
	StartDate         time.Time `json:"start_date"`
	EndDate           time.Time `json:"end_date"`
	Status            string    `json:"status"`
	NotificationSent  bool      `json:"notification_sent"`  // 是否已发送通知
	RenewalPreference string    `json:"renewal_preference"` // yes, no, undecided
}

type Payment struct {
	ID             int64     `json:"id"`
	UserID         int64     `json:"user_id"`
	SubscriptionID int64     `json:"subscription_id"`
	Amount         float64   `json:"amount"`
	PaymentDate    time.Time `json:"payment_date"`
	Status         string    `json:"status"`
	Type           string    `json:"type"` // initial(首次订阅) 或 renewal(续订)
}

type Notification struct {
	ID             int64     `json:"id"`
	UserID         int64     `json:"user_id"`
	SubscriptionID int64     `json:"subscription_id"`
	Type           string    `json:"type"` // 通知类型：expiration_notice, renewal_confirmation等
	Content        string    `json:"content"`
	SentAt         time.Time `json:"sent_at"`
	Status         string    `json:"status"` // sent, failed
}

// Cache 缓存结构
type Cache struct {
	mutex                 sync.RWMutex
	totalUsers            int
	totalPaymentAmount    float64
	activeSubscriptions   int
	newSubscriptionsMonth int     // 本月新增订阅数
	newPaymentAmountMonth float64 // 本月新增付费金额
	renewalsMonth         int     // 本月续订数
	renewalAmountMonth    float64 // 本月续订金额
	lastUpdated           time.Time
}

// 订阅创建请求
type SubscriptionRequest struct {
	UserID int64   `json:"user_id"`
	Plan   string  `json:"plan"`
	Amount float64 `json:"amount"`
}

// 续订请求
type RenewalRequest struct {
	SubscriptionID int64   `json:"subscription_id"`
	UserID         int64   `json:"user_id"`
	Amount         float64 `json:"amount"`
}

// 取消续订请求
type CancelRenewalRequest struct {
	SubscriptionID int64 `json:"subscription_id"`
	UserID         int64 `json:"user_id"`
}

// 系统状态响应
type SystemStats struct {
	TotalUsers            int       `json:"total_users"`
	TotalPaymentAmount    float64   `json:"total_payment_amount"`
	ActiveSubscriptions   int       `json:"active_subscriptions"`
	NewSubscriptionsMonth int       `json:"new_subscriptions_month"`
	NewPaymentAmountMonth float64   `json:"new_payment_amount_month"`
	RenewalsMonth         int       `json:"renewals_month"`
	RenewalAmountMonth    float64   `json:"renewal_amount_month"`
	LastUpdated           time.Time `json:"last_updated"`
}

// 时间段查询请求
type TimeRangeQuery struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// 时间段统计结果
type TimeRangeStats struct {
	PaidUsers     int       `json:"paid_users"`     // 付费用户数
	TotalPayments float64   `json:"total_payments"` // 付费总金额
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
}
