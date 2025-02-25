// 定义主包
package main

// 导入依赖包
import (
	"bufio"   // 缓冲IO读写
	"fmt"     // 格式化IO
	"os"      // 操作系统功能
	"runtime" // 运行时信息
	"sort"    // 排序功能
	"sync"    // 并发控制
	"time"
	"unicode" // Unicode字符处理
)

// 定义单词计数结构体
type WordCount struct {
	Word  string // 单词
	Count int    // 出现次数
}

func main() {
	// 打开文件
	start := time.Now()
	file, err := os.Open("/Users/anker/file_test.txt")
	if err != nil {
		fmt.Printf("无法打开文件: %v\n", err)
		return
	}
	defer file.Close() // 确保文件关闭

	// 初始化并发参数
	numWorkers := runtime.NumCPU()       // 使用CPU核心数作为工作协程数
	lines := make(chan string, 1000)     // 带缓冲的行通道（减少IO等待）
	results := make(chan map[string]int) // 结果收集通道

	var wg sync.WaitGroup // 协程同步控制器

	// 启动工作协程池
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go lineWorker(lines, results, &wg)
	}

	// 启动结果合并协程
	resultChan := make(chan []WordCount)
	go func() {
		resultChan <- processResults(results, numWorkers)
	}()

	// 使用缓冲扫描器读取文件
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 1024*1024) // 初始化1MB缓冲区
	scanner.Buffer(buf, 10*1024*1024) // 设置最大行长度为10MB

	// 逐行读取并发送到通道
	for scanner.Scan() {
		lines <- scanner.Text()
	}
	close(lines) // 关闭通道触发工作协程结束

	// 等待所有工作协程完成
	wg.Wait()
	close(results) // 关闭结果通道

	// 获取排序后的结果并输出
	sorted := <-resultChan
	printTopWords(sorted, 30)
	end := time.Now()
	fmt.Println(end.Sub(start))
}

// 行处理工作协程，只能接受的通道lines，只能发送的通道res
func lineWorker(lines <-chan string, results chan<- map[string]int, wg *sync.WaitGroup) {
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

// 自定义分词函数
func splitToWords(line string) []string {
	var words []string
	var word []rune // 使用rune处理Unicode

	// 遍历每个字符,遇到空格或者其他不是字母的字符，就把之前存入的提交
	for _, r := range line {
		if unicode.IsLetter(r) { // 判断是否为字母
			word = append(word, unicode.ToLower(r)) // 转换为小写
		} else {
			if len(word) > 0 { // 遇到非字母字符时提交单词
				words = append(words, string(word))
				word = nil // 重置缓冲区
			}
		}
	}

	// 处理行末的最后一个单词
	if len(word) > 0 {
		words = append(words, string(word))
	}

	return words
}

// 合并处理结果 仅接受
func processResults(results <-chan map[string]int, workers int) []WordCount {
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

// 排序函数
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
