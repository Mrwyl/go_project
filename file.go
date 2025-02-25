// 定义主包
package main

// 导入依赖包
import (
	"bufio" // 缓冲IO读写
	"fmt"   // 格式化IO
	"log"
	"os"      // 操作系统功能
	"runtime" // 运行时信息
	"sort"    // 排序功能
	"strings"
	"sync" // 并发控制
	"time"
	"unicode" // Unicode字符处理
)

// 定义单词计数结构体
type WordCount struct {
	Word  string // 单词
	Count int    // 出现次数
}

func main() {
	PrintMemUsage("开始读取文件前")
	start := time.Now()
	var wg sync.WaitGroup                        // 协程同步控制器
	lines := make(chan string, 100000)           // 带缓冲的行通道（减少IO等待）
	results := make(chan map[string]int, 100000) // 结果收集通道
	resultChan := make(chan []WordCount, 100000)
	buf := make([]byte, 0, 1024*1024) // 初始化1MB缓冲区

	// 打开文件
	file, err := os.OpenFile("/Users/anker/text_data.txt", os.O_RDONLY, 0) //openfile指令,改为只读

	if err != nil {
		fmt.Printf("无法打开文件: %v\n", err)
		return
	}
	// 立即设置defer并处理关闭错误
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("Error closing file: %v", closeErr)
		}
	}() // 如果移动到os.Open后，会导致空指针崩溃风险：os.Open返回错误时，file是nil。因此在错误处理之前执行defer file.Close()的话，如果err存在，file是nil，那么defer会调用nil.Close()，导致运行时错误。
	// 初始化并发参数
	numWorkers := runtime.NumCPU() // 使用CPU核心数作为工作协程数

	// 启动工作协程池
	for i := 0; i < numWorkers*2; i++ {
		wg.Add(1)
		go countWordsWorker(lines, results, &wg)
	}

	// 启动结果合并协程

	go func() {
		resultChan <- aggregateWordCounts(results, numWorkers)
	}()

	// 使用缓冲扫描器读取文件
	scanner := bufio.NewScanner(file)
	scanner.Buffer(buf, 10*1024*1024) // 设置最大行长度为10MB

	// 逐行读取并发送到通道
	for scanner.Scan() {
		lines <- scanner.Text()
	}

	//AI说这两个close是安全的，不会引发panic
	close(lines) // 关闭通道触发工作协程结束
	// 等待所有工作协程完成
	wg.Wait()
	close(results) // 关闭结果通道

	// 获取排序后的结果并输出
	sorted := <-resultChan
	PrintMemUsage("结果排序完毕后")
	printTopWords(sorted, 30)
	end := time.Now()
	fmt.Println(end.Sub(start))
}

// 行处理工作协程，只能接受的通道lines，只能发送的通道res
func countWordsWorker(lines <-chan string, results chan<- map[string]int, wg *sync.WaitGroup) {
	defer wg.Done()
	localCount := make(map[string]int) // 本地计数器

	// 从通道持续接收行数据
	for line := range lines {
		words := splitToWords(line) // 分词处理
		for _, word := range words {
			localCount[word]++ // 统计单词
		}
	}

	// 发送本地统计结果
	results <- localCount
}

// 分词
func splitToWords(line string) []string {
	fields := strings.Fields(line)
	idx := 0
	// words := make([]string, 0, len(fields)) // 预分配容量
	for _, word := range fields {
		// 使用索引操作代替多次append  AI润色的结果
		clean := strings.TrimFunc(word, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		})
		if clean != "" {
			fields[idx] = clean
			idx++
		}
	}
	return fields[:idx]
}

// 合并协程
func aggregateWordCounts(results <-chan map[string]int, workers int) []WordCount {
	total := make(map[string]int) // 全局计数器

	// 合并所有工作协程的结果
	for i := 0; i < workers; i++ {
		localCount := <-results
		for word, count := range localCount {
			total[word] += count
		}
	}

	return sortWordCounts(total) // 返回排序结果
}

// 排序函数      时间复杂度为O(n log n)
func sortWordCounts(counts map[string]int) []WordCount {
	sorted := make([]WordCount, 0, len(counts))

	// 将map转换为切片
	for word, count := range counts {
		sorted = append(sorted, WordCount{word, count})
	}

	// 自定义排序规则
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Count == sorted[j].Count {
			return sorted[i].Word < sorted[j].Word // 相同频率按字母排序
		}
		return sorted[i].Count > sorted[j].Count // 降序排列
	})

	return sorted
}

// 输出前N个结果
func printTopWords(sorted []WordCount, n int) {
	if n > len(sorted) {
		n = len(sorted)
	}
	// 格式化输出表头
	fmt.Printf("%-6s %-20s %s\n", "排名", "单词", "频率")
	fmt.Println("------------------------------")

	// 输出前N个结果
	for i := 0; i < n; i++ {
		fmt.Printf("%-6d %-20s %d\n", i+1, sorted[i].Word, sorted[i].Count)
	}
}

// PrintMemUsage 用于打印当前 Go 运行时的内存使用情况。
// 参数 tag 可以传入一个标识字符串，方便在日志中区分不同阶段的内存状态。
func PrintMemUsage(tag string) {
	var m runtime.MemStats
	// 读取当前内存统计信息，存入 m 中
	runtime.ReadMemStats(&m)

	// 使用日志打印相关字段：
	// - Alloc：当前堆上已分配且仍在使用的内存总量（字节）
	// - TotalAlloc：程序启动至今分配过的内存总量（含已释放部分）
	// - Sys：Go 运行时向操作系统申请的总内存
	// - NumGC：垃圾回收 (GC) 运行的次数
	log.Printf("[%s] Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n",
		tag,
		bToMb(m.Alloc),
		bToMb(m.TotalAlloc),
		bToMb(m.Sys),
		m.NumGC,
	)
}

// bToMb 将字节数转换为兆字节（MiB）
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
