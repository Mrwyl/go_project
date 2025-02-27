package main

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// DatabaseService 数据库服务
type DatabaseService struct {
	db *sql.DB
}

func NewDatabaseService(dsn string) (*DatabaseService, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(100)          // 最大连接数
	db.SetMaxIdleConns(20)           // 最大空闲连接数
	db.SetConnMaxLifetime(time.Hour) // 连接最长生命周期

	// 验证连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接验证失败: %w", err)
	}

	return &DatabaseService{db: db}, nil
}

// 创建用户
func (s *DatabaseService) CreateUser(user *User) (int64, error) {
	query := `INSERT INTO users (name, email) VALUES (?, ?)`

	result, err := s.db.Exec(query, user.Name, user.Email)
	if err != nil {
		return 0, fmt.Errorf("创建用户失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取用户ID失败: %w", err)
	}

	return id, nil
}

// 用户查询相关方法
func (s *DatabaseService) GetUserByID(id int64) (*User, error) {
	query := `SELECT id, name, email, created_at FROM users WHERE id = ?`

	var user User
	err := s.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	return &user, nil
}

// 获取用户订阅
func (s *DatabaseService) GetUserSubscriptions(userID int64) ([]Subscription, error) {
	query := `SELECT id, user_id, plan, start_date, end_date, status, notification_sent, renewal_preference 
              FROM subscriptions WHERE user_id = ?`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户订阅失败: %w", err)
	}
	defer rows.Close()

	var subscriptions []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(
			&sub.ID,
			&sub.UserID,
			&sub.Plan,
			&sub.StartDate,
			&sub.EndDate,
			&sub.Status,
			&sub.NotificationSent,
			&sub.RenewalPreference,
		); err != nil {
			return nil, fmt.Errorf("解析订阅数据失败: %w", err)
		}
		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, nil
}

// 获取用户当前活跃订阅
func (s *DatabaseService) GetActiveSubscription(userID int64) (*Subscription, error) {
	query := `SELECT id, user_id, plan, start_date, end_date, status, notification_sent, renewal_preference 
             FROM subscriptions 
             WHERE user_id = ? AND (status = ? OR status = ?) 
             ORDER BY end_date DESC LIMIT 1`

	var sub Subscription
	err := s.db.QueryRow(query, userID, StatusSubscribed, StatusRenewed).Scan(
		&sub.ID,
		&sub.UserID,
		&sub.Plan,
		&sub.StartDate,
		&sub.EndDate,
		&sub.Status,
		&sub.NotificationSent,
		&sub.RenewalPreference,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // 用户没有活跃订阅
		}
		return nil, fmt.Errorf("获取活跃订阅失败: %w", err)
	}

	return &sub, nil
}

// 获取需要发送通知的即将到期订阅（未发送通知且3天内到期）
func (s *DatabaseService) GetExpiringSubscriptionsForNotification() ([]Subscription, error) {
	// 获取3天内到期且未发送通知的订阅
	threedays := time.Now().AddDate(0, 0, 3)
	query := `SELECT id, user_id, plan, start_date, end_date, status, notification_sent, renewal_preference 
              FROM subscriptions 
              WHERE end_date <= ? AND end_date > NOW() 
              AND (status = ? OR status = ?) AND notification_sent = false`

	rows, err := s.db.Query(query, threedays, StatusSubscribed, StatusRenewed)
	if err != nil {
		return nil, fmt.Errorf("获取即将到期订阅失败: %w", err)
	}
	defer rows.Close()

	var subscriptions []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(
			&sub.ID,
			&sub.UserID,
			&sub.Plan,
			&sub.StartDate,
			&sub.EndDate,
			&sub.Status,
			&sub.NotificationSent,
			&sub.RenewalPreference,
		); err != nil {
			return nil, fmt.Errorf("解析订阅数据失败: %w", err)
		}
		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, nil
}

// 获取需要更新状态的订阅
func (s *DatabaseService) GetExpiredSubscriptions() ([]Subscription, error) {
	// 获取已过期的订阅
	query := `SELECT id, user_id, plan, start_date, end_date, status, notification_sent, renewal_preference 
              FROM subscriptions 
              WHERE end_date < NOW() 
              AND (status = ? OR status = ?)`

	rows, err := s.db.Query(query, StatusSubscribed, StatusUnsubscribed)
	if err != nil {
		return nil, fmt.Errorf("获取已过期订阅失败: %w", err)
	}
	defer rows.Close()

	var subscriptions []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(
			&sub.ID,
			&sub.UserID,
			&sub.Plan,
			&sub.StartDate,
			&sub.EndDate,
			&sub.Status,
			&sub.NotificationSent,
			&sub.RenewalPreference,
		); err != nil {
			return nil, fmt.Errorf("解析订阅数据失败: %w", err)
		}
		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, nil
}

// 更新订阅状态
func (s *DatabaseService) UpdateSubscriptionStatus(id int64, status string) error {
	query := `UPDATE subscriptions SET status = ? WHERE id = ?`

	_, err := s.db.Exec(query, status, id)
	if err != nil {
		return fmt.Errorf("更新订阅状态失败: %w", err)
	}

	return nil
}

// 更新订阅通知状态
func (s *DatabaseService) UpdateSubscriptionNotificationSent(id int64, sent bool) error {
	query := `UPDATE subscriptions SET notification_sent = ? WHERE id = ?`

	_, err := s.db.Exec(query, sent, id)
	if err != nil {
		return fmt.Errorf("更新订阅通知状态失败: %w", err)
	}

	return nil
}

// 更新订阅续订偏好
func (s *DatabaseService) UpdateRenewalPreference(id int64, preference string) error {
	query := `UPDATE subscriptions SET renewal_preference = ? WHERE id = ?`

	_, err := s.db.Exec(query, preference, id)
	if err != nil {
		return fmt.Errorf("更新续订偏好失败: %w", err)
	}

	return nil
}

// 获取用户付款记录
func (s *DatabaseService) GetUserPayments(userID int64) ([]Payment, error) {
	query := `SELECT id, user_id, subscription_id, amount, payment_date, status, type
              FROM payments WHERE user_id = ?`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户付款记录失败: %w", err)
	}
	defer rows.Close()

	var payments []Payment
	for rows.Next() {
		var payment Payment
		if err := rows.Scan(
			&payment.ID,
			&payment.UserID,
			&payment.SubscriptionID,
			&payment.Amount,
			&payment.PaymentDate,
			&payment.Status,
			&payment.Type,
		); err != nil {
			return nil, fmt.Errorf("解析付款数据失败: %w", err)
		}
		payments = append(payments, payment)
	}

	return payments, nil
}

// 获取特定订阅
func (s *DatabaseService) GetSubscriptionByID(id int64) (*Subscription, error) {
	query := `SELECT id, user_id, plan, start_date, end_date, status, notification_sent, renewal_preference 
              FROM subscriptions WHERE id = ?`

	var sub Subscription
	err := s.db.QueryRow(query, id).Scan(
		&sub.ID,
		&sub.UserID,
		&sub.Plan,
		&sub.StartDate,
		&sub.EndDate,
		&sub.Status,
		&sub.NotificationSent,
		&sub.RenewalPreference,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("订阅不存在")
		}
		return nil, fmt.Errorf("获取订阅失败: %w", err)
	}

	return &sub, nil
}

// 更新订阅日期
func (s *DatabaseService) UpdateSubscriptionDates(id int64, startDate, endDate time.Time) error {
	query := `UPDATE subscriptions SET start_date = ?, end_date = ? WHERE id = ?`

	_, err := s.db.Exec(query, startDate, endDate, id)
	if err != nil {
		return fmt.Errorf("更新订阅日期失败: %w", err)
	}

	return nil
}

// 统计方法 - 用户总数
func (s *DatabaseService) GetTotalUserCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("获取用户总数失败: %w", err)
	}
	return count, nil
}

// 统计方法 - 付款总金额
func (s *DatabaseService) GetTotalPaymentAmount() (float64, error) {
	var total float64
	err := s.db.QueryRow(
		"SELECT COALESCE(SUM(amount), 0) FROM payments WHERE status = 'success'",
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("获取付款总额失败: %w", err)
	}
	return total, nil
}

// 统计方法 - 获取活跃订阅数量
func (s *DatabaseService) GetActiveSubscriptionsCount() (int, error) {
	query := `SELECT COUNT(*) FROM subscriptions 
              WHERE status IN (?, ?)`

	var count int
	err := s.db.QueryRow(query, StatusSubscribed, StatusRenewed).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("获取活跃订阅数失败: %w", err)
	}

	return count, nil
}

// 新增: 获取本月新增订阅数
// func (s *DatabaseService) GetNewSubscriptionsMonth() (int, error) {
//     // 获取本月第一天
//     now := time.Now()
//     firstDayOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

//     query := `SELECT COUNT(*) FROM subscriptions
//               WHERE start_date >= ? AND type = 'initial'`

//     var count int
//     err := s.db.QueryRow(query, firstDayOfMonth).Scan(&count)
//     if err != nil {
//         return 0, fmt.Errorf("获取本月新增订阅数失败: %w", err)
//     }

//	    return count, nil
//	}
//
// 新增: 获取本月新增订阅数
func (s *DatabaseService) GetNewSubscriptionsMonth() (int, error) {
	// 获取本月第一天
	now := time.Now()
	firstDayOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	query := `SELECT COUNT(*) FROM payments 
              WHERE payment_date >= ? AND status = 'success' AND type = 'initial'`

	var count int
	err := s.db.QueryRow(query, firstDayOfMonth).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("获取本月新增订阅数失败: %w", err)
	}

	return count, nil
}

// 新增: 获取本月新增付费金额
func (s *DatabaseService) GetNewPaymentAmountMonth() (float64, error) {
	// 获取本月第一天
	now := time.Now()
	firstDayOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	query := `SELECT COALESCE(SUM(amount), 0) FROM payments 
              WHERE payment_date >= ? AND status = 'success' AND type = 'initial'`

	var total float64
	err := s.db.QueryRow(query, firstDayOfMonth).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("获取本月新增付费金额失败: %w", err)
	}

	return total, nil
}

// 新增: 获取本月续订数
func (s *DatabaseService) GetRenewalsMonth() (int, error) {
	// 获取本月第一天
	now := time.Now()
	firstDayOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	query := `SELECT COUNT(*) FROM payments 
              WHERE payment_date >= ? AND status = 'success' AND type = 'renewal'`

	var count int
	err := s.db.QueryRow(query, firstDayOfMonth).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("获取本月续订数失败: %w", err)
	}

	return count, nil
}

// 新增: 获取本月续订金额
func (s *DatabaseService) GetRenewalAmountMonth() (float64, error) {
	// 获取本月第一天
	now := time.Now()
	firstDayOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	query := `SELECT COALESCE(SUM(amount), 0) FROM payments 
              WHERE payment_date >= ? AND status = 'success' AND type = 'renewal'`

	var total float64
	err := s.db.QueryRow(query, firstDayOfMonth).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("获取本月续订金额失败: %w", err)
	}

	return total, nil
}

// 新增: 按时间段查询付费用户数和付费金额
func (s *DatabaseService) GetPaymentStatsByTimeRange(start, end time.Time) (*TimeRangeStats, error) {
	// 查询期间内有付费记录的唯一用户数
	userQuery := `SELECT COUNT(DISTINCT user_id) FROM payments 
                  WHERE payment_date >= ? AND payment_date <= ? AND status = 'success'`

	var userCount int
	err := s.db.QueryRow(userQuery, start, end).Scan(&userCount)
	if err != nil {
		return nil, fmt.Errorf("查询时间段内付费用户数失败: %w", err)
	}

	// 查询期间内的付费总金额
	amountQuery := `SELECT COALESCE(SUM(amount), 0) FROM payments 
                    WHERE payment_date >= ? AND payment_date <= ? AND status = 'success'`

	var totalAmount float64
	err = s.db.QueryRow(amountQuery, start, end).Scan(&totalAmount)
	if err != nil {
		return nil, fmt.Errorf("查询时间段内付费总额失败: %w", err)
	}

	return &TimeRangeStats{
		PaidUsers:     userCount,
		TotalPayments: totalAmount,
		StartTime:     start,
		EndTime:       end,
	}, nil
}

// BeginTx 开始事务
func (s *DatabaseService) BeginTx() (*sql.Tx, error) {
	return s.db.Begin()
}

// Close 关闭数据库连接
func (s *DatabaseService) Close() error {
	return s.db.Close()
}
