package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// SubscriptionHandler HTTP处理器
type SubscriptionHandler struct {
	service *SubscriptionService
}

// NewSubscriptionHandler 创建新的HTTP处理器
func NewSubscriptionHandler(service *SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{service: service}
}

// HandleUserSubscriptions 处理用户订阅查询请求
func (h *SubscriptionHandler) HandleUserSubscriptions(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	log.Printf("收到用户订阅查询请求: %s %s", r.Method, r.URL.Path)

	if r.Method != http.MethodGet {
		http.Error(w, "只支持GET请求", http.StatusMethodNotAllowed)
		log.Printf("请求方法不允许: %s", r.Method)
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		http.Error(w, "缺少user_id参数", http.StatusBadRequest)
		log.Printf("缺少必要参数: user_id")
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "user_id格式不正确", http.StatusBadRequest)
		log.Printf("参数格式错误: user_id=%s", userIDStr)
		return
	}

	subscriptions, err := h.service.GetUserSubscriptionInfo(userID)
	if err != nil {
		log.Printf("获取用户订阅失败: %v", err)
		http.Error(w, "获取订阅信息失败", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(subscriptions); err != nil {
		log.Printf("编码响应失败: %v", err)
		http.Error(w, "服务器错误", http.StatusInternalServerError)
	}

	log.Printf("处理用户订阅查询请求完成，耗时: %v", time.Since(start))
}

// HandleUserPayments 处理用户支付记录查询请求
func (h *SubscriptionHandler) HandleUserPayments(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	log.Printf("收到用户支付记录查询请求: %s %s", r.Method, r.URL.Path)

	if r.Method != http.MethodGet {
		http.Error(w, "只支持GET请求", http.StatusMethodNotAllowed)
		log.Printf("请求方法不允许: %s", r.Method)
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		http.Error(w, "缺少user_id参数", http.StatusBadRequest)
		log.Printf("缺少必要参数: user_id")
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "user_id格式不正确", http.StatusBadRequest)
		log.Printf("参数格式错误: user_id=%s", userIDStr)
		return
	}

	payments, err := h.service.GetUserPaymentHistory(userID)
	if err != nil {
		log.Printf("获取用户支付记录失败: %v", err)
		http.Error(w, "获取支付记录失败", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payments); err != nil {
		log.Printf("编码响应失败: %v", err)
		http.Error(w, "服务器错误", http.StatusInternalServerError)
	}

	log.Printf("处理用户支付记录查询请求完成，耗时: %v", time.Since(start))
}

// HandleSystemStats 处理系统统计信息查询请求
func (h *SubscriptionHandler) HandleSystemStats(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	log.Printf("收到系统统计信息查询请求: %s %s", r.Method, r.URL.Path)

	if r.Method != http.MethodGet {
		http.Error(w, "只支持GET请求", http.StatusMethodNotAllowed)
		log.Printf("请求方法不允许: %s", r.Method)
		return
	}

	stats := h.service.GetSystemStats()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("编码响应失败: %v", err)
		http.Error(w, "服务器错误", http.StatusInternalServerError)
	}

	log.Printf("处理系统统计信息查询请求完成，耗时: %v", time.Since(start))
}

// HandleCreateUser 处理创建用户请求
func (h *SubscriptionHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	log.Printf("收到创建用户请求: %s %s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST请求", http.StatusMethodNotAllowed)
		log.Printf("请求方法不允许: %s", r.Method)
		return
	}

	// 解析请求体
	var request struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "无效的请求数据", http.StatusBadRequest)
		log.Printf("解析请求体失败: %v", err)
		return
	}

	userID, err := h.service.CreateUser(request.Name, request.Email)
	if err != nil {
		log.Printf("创建用户失败: %v", err)
		http.Error(w, fmt.Sprintf("创建用户失败: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"user_id": userID,
		"message": "用户创建成功",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("编码响应失败: %v", err)
		http.Error(w, "服务器错误", http.StatusInternalServerError)
	}

	log.Printf("处理创建用户请求完成，耗时: %v", time.Since(start))
}

// HandleActivateSubscription 处理激活订阅请求
func (h *SubscriptionHandler) HandleActivateSubscription(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	log.Printf("收到激活订阅请求: %s %s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST请求", http.StatusMethodNotAllowed)
		log.Printf("请求方法不允许: %s", r.Method)
		return
	}

	// 解析请求体
	var request struct {
		UserID int64  `json:"user_id"`
		Plan   string `json:"plan"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "无效的请求数据", http.StatusBadRequest)
		log.Printf("解析请求体失败: %v", err)
		return
	}

	if request.UserID <= 0 || request.Plan == "" {
		http.Error(w, "缺少必要参数", http.StatusBadRequest)
		log.Printf("缺少必要参数: user_id或plan")
		return
	}

	err := h.service.ActivateSubscription(request.UserID, request.Plan)
	if err != nil {
		log.Printf("激活订阅失败: %v", err)
		http.Error(w, fmt.Sprintf("激活订阅失败: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"message": "订阅激活成功",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("编码响应失败: %v", err)
		http.Error(w, "服务器错误", http.StatusInternalServerError)
	}

	log.Printf("处理激活订阅请求完成，耗时: %v", time.Since(start))
}

// HandleRenewSubscription 处理续订请求
func (h *SubscriptionHandler) HandleRenewSubscription(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	log.Printf("收到续订请求: %s %s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST请求", http.StatusMethodNotAllowed)
		log.Printf("请求方法不允许: %s", r.Method)
		return
	}

	// 解析请求体
	var request RenewalRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "无效的请求数据", http.StatusBadRequest)
		log.Printf("解析请求体失败: %v", err)
		return
	}

	if request.UserID <= 0 || request.SubscriptionID <= 0 {
		http.Error(w, "缺少必要参数", http.StatusBadRequest)
		log.Printf("缺少必要参数: user_id或subscription_id")
		return
	}

	// 设置默认金额（如果请求中没有提供）
	if request.Amount <= 0 {
		request.Amount = SubscriptionPrice
	}

	err := h.service.RenewSubscription(request)
	if err != nil {
		log.Printf("续订失败: %v", err)
		http.Error(w, fmt.Sprintf("续订失败: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"message": "续订成功",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("编码响应失败: %v", err)
		http.Error(w, "服务器错误", http.StatusInternalServerError)
	}

	log.Printf("处理续订请求完成，耗时: %v", time.Since(start))
}

// HandleCancelRenewal 处理取消续订请求
func (h *SubscriptionHandler) HandleCancelRenewal(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	log.Printf("收到取消续订请求: %s %s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST请求", http.StatusMethodNotAllowed)
		log.Printf("请求方法不允许: %s", r.Method)
		return
	}

	// 解析请求体
	var request CancelRenewalRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "无效的请求数据", http.StatusBadRequest)
		log.Printf("解析请求体失败: %v", err)
		return
	}

	if request.UserID <= 0 || request.SubscriptionID <= 0 {
		http.Error(w, "缺少必要参数", http.StatusBadRequest)
		log.Printf("缺少必要参数: user_id或subscription_id")
		return
	}

	err := h.service.CancelRenewal(request)
	if err != nil {
		log.Printf("取消续订失败: %v", err)
		http.Error(w, fmt.Sprintf("取消续订失败: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"message": "取消续订成功",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("编码响应失败: %v", err)
		http.Error(w, "服务器错误", http.StatusInternalServerError)
	}

	log.Printf("处理取消续订请求完成，耗时: %v", time.Since(start))
}

// HandleMonthlyStats 处理月度统计查询请求（新增功能）
func (h *SubscriptionHandler) HandleMonthlyStats(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	log.Printf("收到月度统计查询请求: %s %s", r.Method, r.URL.Path)

	if r.Method != http.MethodGet {
		http.Error(w, "只支持GET请求", http.StatusMethodNotAllowed)
		log.Printf("请求方法不允许: %s", r.Method)
		return
	}

	stats := h.service.GetSystemStats()

	// 提取运营关注的月度统计数据
	monthlyStats := map[string]interface{}{
		"new_subscriptions_month":  stats.NewSubscriptionsMonth,
		"new_payment_amount_month": stats.NewPaymentAmountMonth,
		"renewals_month":           stats.RenewalsMonth,
		"renewal_amount_month":     stats.RenewalAmountMonth,
		"last_updated":             stats.LastUpdated,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(monthlyStats); err != nil {
		log.Printf("编码响应失败: %v", err)
		http.Error(w, "服务器错误", http.StatusInternalServerError)
	}

	log.Printf("处理月度统计查询请求完成，耗时: %v", time.Since(start))
}

// HandleTimeRangeStats 处理时间段统计查询请求（新增功能）
func (h *SubscriptionHandler) HandleTimeRangeStats(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	log.Printf("收到时间段统计查询请求: %s %s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST请求", http.StatusMethodNotAllowed)
		log.Printf("请求方法不允许: %s", r.Method)
		return
	}

	// 解析请求体
	var request TimeRangeQuery
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "无效的请求数据", http.StatusBadRequest)
		log.Printf("解析请求体失败: %v", err)
		return
	}

	// 验证时间范围
	if request.StartTime.IsZero() || request.EndTime.IsZero() {
		http.Error(w, "开始时间和结束时间不能为空", http.StatusBadRequest)
		log.Printf("缺少必要参数: start_time或end_time")
		return
	}

	if request.EndTime.Before(request.StartTime) {
		http.Error(w, "结束时间不能早于开始时间", http.StatusBadRequest)
		log.Printf("参数错误: end_time早于start_time")
		return
	}

	stats, err := h.service.GetPaymentStatsByTimeRange(request)
	if err != nil {
		log.Printf("查询时间段统计失败: %v", err)
		http.Error(w, fmt.Sprintf("查询统计失败: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("编码响应失败: %v", err)
		http.Error(w, "服务器错误", http.StatusInternalServerError)
	}

	log.Printf("处理时间段统计查询请求完成，耗时: %v", time.Since(start))
}
