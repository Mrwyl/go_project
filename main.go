package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// 系统配置
type Config struct {
	DatabaseDSN string
	ServerPort  int
	LogFile     string
}

// 加载配置（在实际应用中通常从环境变量或配置文件中加载）
func loadConfig() *Config {
	// 这里为了演示简化，使用硬编码的配置
	return &Config{
		DatabaseDSN: "root:181900@tcp(127.0.0.1:3306)/subscription_test_db?parseTime=true",
		ServerPort:  8080,
		LogFile:     "subscription_service.log",
	}
}

// 初始化日志
func initLogger(logFile string) {
	// 如果指定了日志文件，则同时输出到文件和标准输出
	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Printf("无法打开日志文件: %v，将只使用标准输出", err)
		} else {
			log.SetOutput(file)
			log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile | log.LUTC)
			log.Println("日志初始化完成，输出到文件:", logFile)
		}
	}
}

func main() {
	// 加载配置
	config := loadConfig()

	// 初始化日志
	initLogger(config.LogFile)

	log.Println("订阅系统服务正在启动...")

	// 创建订阅服务
	service, err := NewSubscriptionService(config.DatabaseDSN)
	if err != nil {
		log.Fatalf("创建订阅服务失败: %v", err)
	}

	// 启动任务调度器
	scheduler := NewTaskScheduler(service)
	scheduler.Start()

	// 创建HTTP处理器
	handler := NewSubscriptionHandler(service)

	// 注册API路由
	mux := http.NewServeMux()

	// 用户相关API
	mux.HandleFunc("/api/subscriptions", handler.HandleUserSubscriptions)
	mux.HandleFunc("/api/payments", handler.HandleUserPayments)
	mux.HandleFunc("/api/users", handler.HandleCreateUser)
	mux.HandleFunc("/api/subscriptions/activate", handler.HandleActivateSubscription)
	mux.HandleFunc("/api/subscriptions/renew", handler.HandleRenewSubscription)
	mux.HandleFunc("/api/subscriptions/cancel", handler.HandleCancelRenewal)

	// 管理相关API
	mux.HandleFunc("/api/admin/stats", handler.HandleSystemStats)
	mux.HandleFunc("/api/admin/monthly-stats", handler.HandleMonthlyStats)
	mux.HandleFunc("/api/admin/time-range-stats", handler.HandleTimeRangeStats)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.ServerPort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 创建一个通道来接收终止信号
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 启动HTTP服务器
	go func() {
		log.Printf("HTTP服务器启动，监听端口: %d", config.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP服务器启动失败: %v", err)
		}
	}()

	// 优雅关闭
	go func() {
		<-quit
		log.Println("订阅系统服务收到终止信号，准备关闭...")

		// 首先停止接收新的请求
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("HTTP服务器强制关闭: %v", err)
		}

		// 停止任务调度器
		scheduler.Stop()

		// 关闭服务
		if err := service.Close(); err != nil {
			log.Printf("关闭订阅服务时发生错误: %v", err)
		}

		close(done)
	}()

	// 等待服务正常关闭
	<-done
	log.Println("订阅系统服务已成功关闭")
}
