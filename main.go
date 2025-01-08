package main

import (
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "sync"
    "time"
)

const (
    concurrentDownloads = 50
    userAgent          = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.2903.9"
)

var urls = []string{
    "https://dev.iw233.cn/api.php?sort=random",
    "https://api.iw233.cn/api.php?sort=random",
    "https://iw233.cn/api.php?sort=random",
}

func createHTTPClient() *http.Client {
    transport := &http.Transport{
        MaxIdleConns:        100,              // 增加空闲连接数
        MaxIdleConnsPerHost: 100,              // 增加每个主机的最大空闲连接数
        IdleConnTimeout:     90 * time.Second, // 空闲连接超时时间
        DisableCompression:  true,             // 禁用压缩可能提高大文件下载速度
        // 开启长连接
        DisableKeepAlives: false,
    }

    return &http.Client{
        Transport: transport,
        Timeout:   30 * time.Second,
    }
}

func main() {
    // Parse command line arguments
    if len(os.Args) != 3 {
        fmt.Println("Usage:", os.Args[0], "<folder> <number>")
        os.Exit(1)
    }

    folder := os.Args[1]
    var number int
    fmt.Sscanf(os.Args[2], "%d", &number)

    startTime := time.Now()

    // Create folder if it doesn't exist
    if err := os.MkdirAll(folder, 0755); err != nil {
        fmt.Printf("Failed to create folder: %v\n", err)
        os.Exit(1)
    }

    // Find fastest URL
    fastestURL := findFastestURL()
    if fastestURL == "" {
        fmt.Println("No available site found. Exiting.")
        os.Exit(1)
    }

    fmt.Printf("Using the fastest site: %s\n", fastestURL)

    // 创建优化后的HTTP客户端
    client := createHTTPClient()

    // 创建工作池
    jobs := make(chan int, number)
    results := make(chan error, number)

    // 启动工作协程
    for w := 1; w <= concurrentDownloads; w++ {
        go worker(w, jobs, results, client, folder, fastestURL)
    }

    // 发送任务
    for i := 1; i <= number; i++ {
        jobs <- i
    }
    close(jobs)

    // 收集结果
    for i := 1; i <= number; i++ {
        if err := <-results; err != nil {
            fmt.Printf("Error downloading image: %v\n", err)
        }
    }

    duration := time.Since(startTime)
    minutes := int(duration.Minutes())
    seconds := int(duration.Seconds()) % 60
    fmt.Printf("Done. Downloaded %d images to %s, use to %dmin%ds.\n", 
        number, folder, minutes, seconds)
}

func findFastestURL() string {
    var fastestURL string
    minTime := float64(999999)

    client := &http.Client{
        Timeout: 5 * time.Second,
    }

    for _, url := range urls {
        // Test connection time
        start := time.Now()
        req, err := http.NewRequest("HEAD", url, nil)
        if err != nil {
            continue
        }

        req.Header.Set("User-Agent", userAgent)
        req.Header.Set("Referer", "https://www.baidu.com/s?wd=iw233")

        resp, err := client.Do(req)
        if err != nil {
            continue
        }
        resp.Body.Close()

        if resp.StatusCode == http.StatusForbidden {
            fmt.Printf("Site %s is forbidden (HTTP 403). Skipping.\n", url)
            continue
        }

        connectionTime := time.Since(start).Seconds()
        if connectionTime < minTime {
            minTime = connectionTime
            fastestURL = url
        }
    }

    return fastestURL
}

func downloadImage(client *http.Client, url, filename string) error {
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return err
    }

    req.Header.Set("User-Agent", userAgent)
    req.Header.Set("Referer", "https://weibo.com/")
    req.Header.Set("Accept-Language", "zh-CN,cn;q=0.9")

    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("server returned status code %d", resp.StatusCode)
    }

    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    // 使用更大的缓冲区进行拷贝
    buf := make([]byte, 32*1024) // 32KB buffer
    _, err = io.CopyBuffer(file, resp.Body, buf)
    return err
}

func worker(id int, jobs <-chan int, results chan<- error, client *http.Client, folder, url string) {
    for i := range jobs {
        filename := filepath.Join(folder, fmt.Sprintf("%d.jpg", time.Now().UnixNano()))
        err := downloadImage(client, url, filename)
        results <- err
        if err == nil {
            fmt.Printf("Worker %d downloaded image %d\n", id, i)
        }
    }
}
