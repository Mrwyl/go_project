package main

import (
	"database/sql"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// 测试数据库配置
const testDSN = "root:181900@tcp(127.0.0.1:3306)/subscription_test_db?parseTime=true"

// 测试前准备测试数据库
func setupTestDB() error {
	db, err := sql.Open("mysql", testDSN)
	if err != nil {
		return err
	}
	defer db.Close()

	// 清空测试数据
	tables := []string{"notifications", "payments", "subscriptions", "users"}
	// for _, table := range tables {
	// 	_, err := db.Exec("TRUNCATE TABLE " + table)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	for _, table := range tables {
		_, err := db.Exec("DELETE FROM " + table)
		if err != nil {
			return err
		}
		// 如果需要重置自增主键
		_, err = db.Exec("ALTER TABLE " + table + " AUTO_INCREMENT = 1;")
		if err != nil {
			return err
		}
	}
	return nil

}

// 测试前初始化
func TestMain(m *testing.M) {
	// 设置日志
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// 准备测试数据库
	if err := setupTestDB(); err != nil {
		log.Fatalf("设置测试数据库失败: %v", err)
	}

	// 运行测试
	exitCode := m.Run()

	// 完成后退出
	os.Exit(exitCode)
}

// 创建测试服务实例
func createTestService(t *testing.T) *SubscriptionService {
	service, err := NewSubscriptionService(testDSN)
	if err != nil {
		t.Fatalf("创建订阅服务失败: %v", err)
	}
	return service
}

// 测试用户创建功能
func TestCreateUser(t *testing.T) {
	// 创建服务实例
	service := createTestService(t)
	defer service.Close()

	// 测试数据
	testCases := []struct {
		name     string
		userName string
		email    string
		wantErr  bool
	}{
		{
			name:     "正常创建用户",
			userName: "测试用户",
			email:    "test@example.com",
			wantErr:  false,
		},
		{
			name:     "空用户名",
			userName: "",
			email:    "invalid@example.com",
			wantErr:  true,
		},
		{
			name:     "空邮箱",
			userName: "无邮箱用户",
			email:    "",
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			userID, err := service.CreateUser(tc.userName, tc.email)

			// 检查错误
			if (err != nil) != tc.wantErr {
				t.Errorf("CreateUser() 错误 = %v, 期望错误 = %v", err, tc.wantErr)
				return
			}

			// 如果期望成功，验证用户ID是否有效
			if !tc.wantErr {
				if userID <= 0 {
					t.Errorf("创建用户返回无效ID: %d", userID)
				}

				// 尝试获取用户验证是否创建成功
				user, err := service.db.GetUserByID(userID)
				if err != nil {
					t.Errorf("创建后无法获取用户: %v", err)
				}

				if user.Name != tc.userName || user.Email != tc.email {
					t.Errorf("创建的用户数据不匹配: 期望 name=%s, email=%s; 实际 name=%s, email=%s",
						tc.userName, tc.email, user.Name, user.Email)
				}

				// 检查是否创建了未激活订阅
				subs, err := service.db.GetUserSubscriptions(userID)
				if err != nil {
					t.Errorf("获取用户订阅失败: %v", err)
				}

				if len(subs) != 1 || subs[0].Status != StatusInactive {
					t.Errorf("用户未创建未激活订阅或状态错误: %+v", subs)
				}
			}
		})
	}
}

// 测试激活订阅
func TestActivateSubscription(t *testing.T) {
	// 创建服务实例
	service := createTestService(t)
	defer service.Close()

	// 创建测试用户
	testUser := "订阅测试用户"
	testEmail := "subscription_test@example.com"
	userID, err := service.CreateUser(testUser, testEmail)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	// 测试激活订阅
	err = service.ActivateSubscription(userID, "premium")
	if err != nil {
		t.Errorf("激活订阅失败: %v", err)
	}

	// 验证订阅状态
	subs, err := service.db.GetUserSubscriptions(userID)
	if err != nil {
		t.Errorf("获取用户订阅失败: %v", err)
	}

	if len(subs) != 1 {
		t.Fatalf("期望1个订阅，实际有%d个", len(subs))
	}

	if subs[0].Status != StatusSubscribed {
		t.Errorf("订阅状态错误: 期望=%s, 实际=%s", StatusSubscribed, subs[0].Status)
	}

	if subs[0].Plan != "premium" {
		t.Errorf("订阅计划错误: 期望=premium, 实际=%s", subs[0].Plan)
	}

	// 验证结束日期是否为1个月后
	expectedEndDate := time.Now().AddDate(0, 1, 0)
	daysDiff := subs[0].EndDate.Sub(expectedEndDate).Hours() / 24
	if daysDiff < -1 || daysDiff > 1 { // 允许1天误差
		t.Errorf("订阅结束日期错误: 实际=%v, 期望接近=%v", subs[0].EndDate, expectedEndDate)
	}

	// 验证是否创建了支付记录
	payments, err := service.db.GetUserPayments(userID)
	if err != nil {
		t.Errorf("获取用户付款记录失败: %v", err)
	}

	if len(payments) != 1 {
		t.Fatalf("期望1条付款记录，实际有%d条", len(payments))
	}

	if payments[0].Amount != SubscriptionPrice {
		t.Errorf("付款金额错误: 期望=%.2f, 实际=%.2f", SubscriptionPrice, payments[0].Amount)
	}

	if payments[0].Type != "initial" {
		t.Errorf("付款类型错误: 期望=initial, 实际=%s", payments[0].Type)
	}
}

// 测试续订功能
func TestRenewSubscription(t *testing.T) {
	// 创建服务实例
	service := createTestService(t)
	defer service.Close()

	// 创建并激活测试用户订阅
	testUser := "续订测试用户"
	testEmail := "renewal_test@example.com"
	userID, err := service.CreateUser(testUser, testEmail)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	err = service.ActivateSubscription(userID, "basic")
	if err != nil {
		t.Fatalf("激活订阅失败: %v", err)
	}

	// 获取订阅ID
	subs, err := service.db.GetUserSubscriptions(userID)
	if err != nil || len(subs) != 1 {
		t.Fatalf("获取用户订阅失败: %v", err)
	}

	subID := subs[0].ID

	// 测试续订
	request := RenewalRequest{
		SubscriptionID: subID,
		UserID:         userID,
		Amount:         SubscriptionPrice,
	}

	err = service.RenewSubscription(request)
	if err != nil {
		t.Errorf("续订失败: %v", err)
	}

	// 验证订阅状态
	subs, err = service.db.GetUserSubscriptions(userID)
	if err != nil {
		t.Errorf("获取用户订阅失败: %v", err)
	}

	if len(subs) != 1 || subs[0].Status != StatusRenewed {
		t.Errorf("续订后状态错误: 期望=%s, 实际=%s", StatusRenewed, subs[0].Status)
	}

	// 验证续订偏好
	if subs[0].RenewalPreference != "yes" {
		t.Errorf("续订偏好错误: 期望=yes, 实际=%s", subs[0].RenewalPreference)
	}

	// 验证付款记录
	payments, err := service.db.GetUserPayments(userID)
	if err != nil {
		t.Errorf("获取用户付款记录失败: %v", err)
	}

	if len(payments) != 2 { // 初始付款 + 续订付款
		t.Fatalf("期望2条付款记录，实际有%d条", len(payments))
	}

	var renewalPayment *Payment
	for i := range payments {
		if payments[i].Type == "renewal" {
			renewalPayment = &payments[i]
			break
		}
	}

	if renewalPayment == nil {
		t.Errorf("未找到续订付款记录")
	} else {
		if renewalPayment.Amount != SubscriptionPrice {
			t.Errorf("续订金额错误: 期望=%.2f, 实际=%.2f", SubscriptionPrice, renewalPayment.Amount)
		}
	}
}

// 测试取消续订功能
func TestCancelRenewal(t *testing.T) {
	// 创建服务实例
	service := createTestService(t)
	defer service.Close()

	// 创建并激活测试用户订阅
	testUser := "取消续订测试用户"
	testEmail := "cancel_test@example.com"
	userID, err := service.CreateUser(testUser, testEmail)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	err = service.ActivateSubscription(userID, "basic")
	if err != nil {
		t.Fatalf("激活订阅失败: %v", err)
	}

	// 获取订阅ID
	subs, err := service.db.GetUserSubscriptions(userID)
	if err != nil || len(subs) != 1 {
		t.Fatalf("获取用户订阅失败: %v", err)
	}

	subID := subs[0].ID

	// 测试取消续订
	request := CancelRenewalRequest{
		SubscriptionID: subID,
		UserID:         userID,
	}

	err = service.CancelRenewal(request)
	if err != nil {
		t.Errorf("取消续订失败: %v", err)
	}

	// 验证订阅状态和续订偏好
	subs, err = service.db.GetUserSubscriptions(userID)
	if err != nil {
		t.Errorf("获取用户订阅失败: %v", err)
	}

	if len(subs) != 1 || subs[0].Status != StatusUnsubscribed {
		t.Errorf("取消续订后状态错误: 期望=%s, 实际=%s", StatusUnsubscribed, subs[0].Status)
	}

	if subs[0].RenewalPreference != "no" {
		t.Errorf("续订偏好错误: 期望=no, 实际=%s", subs[0].RenewalPreference)
	}
}

// 测试系统统计
func TestGetSystemStats(t *testing.T) {
	// 创建服务实例
	service := createTestService(t)
	defer service.Close()

	// 获取初始统计
	initialStats := service.GetSystemStats()

	// 创建几个测试用户和订阅
	testUsers := []struct {
		name  string
		email string
	}{
		{"统计测试用户1", "stats_test1@example.com"},
		{"统计测试用户2", "stats_test2@example.com"},
		{"统计测试用户3", "stats_test3@example.com"},
	}

	for _, user := range testUsers {
		userID, err := service.CreateUser(user.name, user.email)
		if err != nil {
			t.Fatalf("创建测试用户失败: %v", err)
		}

		err = service.ActivateSubscription(userID, "basic")
		if err != nil {
			t.Fatalf("激活订阅失败: %v", err)
		}
	}

	// 强制刷新缓存
	if err := service.cache.refreshCache(); err != nil {
		t.Fatalf("刷新缓存失败: %v", err)
	}

	// 获取更新后的统计
	updatedStats := service.GetSystemStats()

	// 验证用户数增加
	expectedUserIncrease := len(testUsers)
	actualUserIncrease := updatedStats.TotalUsers - initialStats.TotalUsers
	if actualUserIncrease != expectedUserIncrease {
		t.Errorf("用户数增加错误: 期望=%d, 实际=%d", expectedUserIncrease, actualUserIncrease)
	}

	// 验证活跃订阅数增加
	expectedSubIncrease := len(testUsers)
	actualSubIncrease := updatedStats.ActiveSubscriptions - initialStats.ActiveSubscriptions
	if actualSubIncrease != expectedSubIncrease {
		t.Errorf("活跃订阅数增加错误: 期望=%d, 实际=%d", expectedSubIncrease, actualSubIncrease)
	}

	// 验证付款金额增加
	expectedAmountIncrease := float64(len(testUsers)) * SubscriptionPrice
	actualAmountIncrease := updatedStats.TotalPaymentAmount - initialStats.TotalPaymentAmount
	if actualAmountIncrease != expectedAmountIncrease {
		t.Errorf("付款总额增加错误: 期望=%.2f, 实际=%.2f", expectedAmountIncrease, actualAmountIncrease)
	}

	// 验证更新时间是否刷新
	if !updatedStats.LastUpdated.After(initialStats.LastUpdated) {
		t.Errorf("统计更新时间未刷新: 初始=%v, 更新后=%v", initialStats.LastUpdated, updatedStats.LastUpdated)
	}
}

// 创建测试数据库连接和通知服务实例
func createTestNotificationService(t *testing.T) (*NotificationService, *DatabaseService) {
	db, err := NewDatabaseService(testDSN)
	if err != nil {
		t.Fatalf("创建数据库服务失败: %v", err)
	}

	notificationSvc := NewNotificationService(db)
	return notificationSvc, db
}

// 创建测试用户和订阅
func createTestUserAndSubscription(t *testing.T, db *DatabaseService) (int64, int64) {
	// 创建测试用户
	user := &User{
		Name:  "通知测试用户",
		Email: "notification_test@example.com",
	}

	userID, err := db.CreateUser(user)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	// 创建测试订阅
	now := time.Now()
	endDate := now.AddDate(0, 1, 0)

	tx, err := db.BeginTx()
	if err != nil {
		t.Fatalf("开始事务失败: %v", err)
	}

	result, err := tx.Exec(
		`INSERT INTO subscriptions 
        (user_id, plan, start_date, end_date, status, notification_sent, renewal_preference) 
        VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID,
		"premium",
		now,
		endDate,
		StatusSubscribed,
		false,
		"undecided",
	)

	if err != nil {
		tx.Rollback()
		t.Fatalf("创建测试订阅失败: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("提交事务失败: %v", err)
	}

	subscriptionID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("获取订阅ID失败: %v", err)
	}

	return userID, subscriptionID
}

// 获取用户最新的通知
func getLatestNotification(t *testing.T, db *DatabaseService, userID int64, notificationType string) *Notification {
	query := `SELECT id, user_id, subscription_id, type, content, sent_at, status 
              FROM notifications 
              WHERE user_id = ? AND type = ? 
              ORDER BY sent_at DESC LIMIT 1`

	var notification Notification
	err := db.db.QueryRow(query, userID, notificationType).Scan(
		&notification.ID,
		&notification.UserID,
		&notification.SubscriptionID,
		&notification.Type,
		&notification.Content,
		&notification.SentAt,
		&notification.Status,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		t.Fatalf("查询通知失败: %v", err)
	}

	return &notification
}

// 测试发送到期通知
func TestSendExpirationNotice(t *testing.T) {
	notificationSvc, db := createTestNotificationService(t)
	defer db.Close()

	userID, subscriptionID := createTestUserAndSubscription(t, db)

	// 发送通知
	err := notificationSvc.SendExpirationNotice(userID, subscriptionID)
	if err != nil {
		t.Fatalf("发送到期通知失败: %v", err)
	}

	// 验证通知记录
	notification := getLatestNotification(t, db, userID, "expiration_notice")
	if notification == nil {
		t.Fatal("未找到通知记录")
	}

	// 检查通知内容是否包含预期的关键信息
	if notification.UserID != userID {
		t.Errorf("通知用户ID不匹配: 期望=%d, 实际=%d", userID, notification.UserID)
	}

	if notification.SubscriptionID != subscriptionID {
		t.Errorf("通知订阅ID不匹配: 期望=%d, 实际=%d", subscriptionID, notification.SubscriptionID)
	}

	if notification.Status != "sent" {
		t.Errorf("通知状态错误: 期望=sent, 实际=%s", notification.Status)
	}

	if !strings.Contains(notification.Content, "到期") {
		t.Errorf("通知内容未包含'到期'关键词: %s", notification.Content)
	}
}

// 测试发送续约成功通知
func TestSendRenewalConfirmation(t *testing.T) {
	notificationSvc, db := createTestNotificationService(t)
	defer db.Close()

	userID, subscriptionID := createTestUserAndSubscription(t, db)

	// 发送通知
	err := notificationSvc.SendRenewalConfirmation(userID, subscriptionID)
	if err != nil {
		t.Fatalf("发送续约确认通知失败: %v", err)
	}

	// 验证通知记录
	notification := getLatestNotification(t, db, userID, "renewal_confirmation")
	if notification == nil {
		t.Fatal("未找到通知记录")
	}

	// 检查通知内容是否包含预期的关键信息
	if !strings.Contains(notification.Content, "成功续约") {
		t.Errorf("通知内容未包含'成功续约'关键词: %s", notification.Content)
	}
}

// 测试发送取消续约确认通知
func TestSendCancelConfirmation(t *testing.T) {
	notificationSvc, db := createTestNotificationService(t)
	defer db.Close()

	userID, subscriptionID := createTestUserAndSubscription(t, db)

	// 发送通知
	err := notificationSvc.SendCancelConfirmation(userID, subscriptionID)
	if err != nil {
		t.Fatalf("发送取消续约通知失败: %v", err)
	}

	// 验证通知记录
	notification := getLatestNotification(t, db, userID, "cancel_confirmation")
	if notification == nil {
		t.Fatal("未找到通知记录")
	}

	// 检查通知内容是否包含预期的关键信息
	if !strings.Contains(notification.Content, "取消续约") {
		t.Errorf("通知内容未包含'取消续约'关键词: %s", notification.Content)
	}
}

// 测试发送订阅结束通知
func TestSendSubscriptionEndedNotice(t *testing.T) {
	notificationSvc, db := createTestNotificationService(t)
	defer db.Close()

	userID, subscriptionID := createTestUserAndSubscription(t, db)

	// 发送通知
	err := notificationSvc.SendSubscriptionEndedNotice(userID, subscriptionID)
	if err != nil {
		t.Fatalf("发送订阅结束通知失败: %v", err)
	}

	// 验证通知记录
	notification := getLatestNotification(t, db, userID, "subscription_ended")
	if notification == nil {
		t.Fatal("未找到通知记录")
	}

	// 检查通知内容是否包含预期的关键信息
	if !strings.Contains(notification.Content, "订阅已结束") {
		t.Errorf("通知内容未包含'订阅已结束'关键词: %s", notification.Content)
	}
}

// 测试处理无效用户ID的情况
func TestSendNotificationInvalidUser(t *testing.T) {
	notificationSvc, db := createTestNotificationService(t)
	defer db.Close()

	_, subscriptionID := createTestUserAndSubscription(t, db)

	// 使用不存在的用户ID
	invalidUserID := int64(9999999)

	// 尝试发送通知
	err := notificationSvc.SendExpirationNotice(invalidUserID, subscriptionID)

	// 预期会失败
	if err == nil {
		t.Fatal("对不存在的用户发送通知应当失败，但却成功了")
	}

	// 验证错误信息
	if !strings.Contains(err.Error(), "用户不存在") {
		t.Errorf("错误消息不符合预期: %v", err)
	}
}

// 测试处理无效订阅ID的情况
func TestSendNotificationInvalidSubscription(t *testing.T) {
	notificationSvc, db := createTestNotificationService(t)
	defer db.Close()

	userID, _ := createTestUserAndSubscription(t, db)

	// 使用不存在的订阅ID
	invalidSubID := int64(9999999)

	// 尝试发送通知
	err := notificationSvc.SendRenewalConfirmation(userID, invalidSubID)

	// 预期会失败
	if err == nil {
		t.Fatal("对不存在的订阅发送通知应当失败，但却成功了")
	}

	// 验证错误信息
	if !strings.Contains(err.Error(), "订阅不存在") {
		t.Errorf("错误消息不符合预期: %v", err)
	}
}
