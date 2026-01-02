package main

import (
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "sync"
    "sync/atomic"
    "time"
    "context"
    "net/url"
)

const (
    concurrentDownloads = 64
    userAgent          = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.2903.9"
    bufferSize        = 1<< 20
)

var urls = []string{
    "https://cnmiw.com/api.php?sort=setu",
    "https://cnmiw.com/api.php?sort=top",
    "https://cnmiw.com/api.php?sort=random",
    //"https://api.iw233.cn/api.php?sort=random",
    //"https://iw233.cn/api.php?sort=random",
}

var baseDownloadReq = func() *http.Request {
    req, _ := http.NewRequest("GET", "", nil)
    req.Header.Set("User-Agent", userAgent)
    req.Header.Set("Referer", "https://weibo.com/ ")
    req.Header.Set("Accept-Language", "zh-CN,cn;q=0.9")
    req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
    req.Header.Set("Connection", "keep-alive")
    return req
}()

// 使用 sync.Pool 来复用缓冲区
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, bufferSize)
    },
}

func main() {
    if len(os.Args) != 3 {
        fmt.Println("Usage:", os.Args[0], "<folder> <number>")
        os.Exit(1)
    }

    folder := os.Args[1]
    var number int
    fmt.Sscanf(os.Args[2], "%d", &number)

    startTime := time.Now()

    if err := os.MkdirAll(folder, 0755); err != nil {
        fmt.Printf("Failed to create folder: %v\n", err)
        os.Exit(1)
    }

    // 优化的测速逻辑，并行测试所有URL
    fastestURL := findFastestURL()
    if fastestURL == "" {
        fmt.Println("No available site found. Exiting.")
        os.Exit(1)
    }
    fmt.Printf("Using the fastest site: %s\n", fastestURL)

    // 创建优化的 HTTP 客户端
    transport := &http.Transport{
        MaxIdleConns:        100,
        MaxConnsPerHost:     100,
        IdleConnTimeout:     90 * time.Second,
        DisableCompression:  true,
        DisableKeepAlives:   false,
        ForceAttemptHTTP2:   true,
        MaxIdleConnsPerHost: 100,
        WriteBufferSize:     bufferSize, // 增加写缓冲
        ReadBufferSize:      bufferSize, // 增加读缓冲
    }

    client := &http.Client{
        Transport: transport,
        Timeout:   30 * time.Second,
    }

    // 使用原子计数器跟踪成功下载数
    var successCount int32
    var fileID int64
    var wg sync.WaitGroup
    sem := make(chan struct{}, concurrentDownloads)

    // 启动下载协程
    for i := 1; i <= number; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            sem <- struct{}{} // 获取信号量
            defer func() { <-sem }() // 释放信号量

            id := atomic.AddInt64(&fileID, 1)
            filename := filepath.Join(folder, fmt.Sprintf("%d.jpg", id))
            if err := downloadWithRetry(client, fastestURL, filename, 3); err == nil {
                atomic.AddInt32(&successCount, 1)
                fmt.Printf("Downloaded %d of %d. filename: %s\n", i, number, filename)
            } else {
                fmt.Printf("Failed to download image %d after retries: %v\n", i, err)
            }
        }(i)
    }

    wg.Wait()

    duration := time.Since(startTime)
    fmt.Printf("Done. Successfully downloaded %d/%d images to %s in %v\n",
        atomic.LoadInt32(&successCount), number, folder, duration)
}

func findFastestURL() string {
    type result struct {
        url      string
        duration time.Duration
        err      error
    }

    results := make(chan result, len(urls))
    var wg sync.WaitGroup

    client := &http.Client{
        Timeout: 5 * time.Second,
        Transport: &http.Transport{
            DisableKeepAlives: false,
            ForceAttemptHTTP2: true,
        },
    }

    for _, url := range urls {
        wg.Add(1)
        go func(url string) {
            defer wg.Done()
            start := time.Now()

            req, err := http.NewRequest("HEAD", url, nil)
            if err != nil {
                results <- result{url: url, err: err}
                return
            }

            req.Header.Set("User-Agent", userAgent)
            req.Header.Set("Referer", "https://www.baidu.com/s?wd=iw233")

            resp, err := client.Do(req)
            if err != nil {
                results <- result{url: url, err: err}
                return
            }
            resp.Body.Close()

            if resp.StatusCode == http.StatusForbidden {
                results <- result{url: url, err: fmt.Errorf("HTTP 403")}
                return
            }

            results <- result{
                url:      url,
                duration: time.Since(start),
                err:     nil,
            }
        }(url)
    }

    // 启动一个协程来关闭结果通道
    go func() {
        wg.Wait()
        close(results)
    }()

    var fastestURL string
    minDuration := time.Hour

    for r := range results {
        if r.err == nil && r.duration < minDuration {
            minDuration = r.duration
            fastestURL = r.url
        }
    }

    return fastestURL
}

func downloadWithRetry(client *http.Client, url, filename string, maxRetries int) error {
    var lastErr error
    for i := 0; i < maxRetries; i++ {
        if i > 0 {
            time.Sleep(time.Duration(i) * time.Second)
        }

        if err := downloadImage(client, url, filename); err != nil {
            lastErr = err
            continue
        }
        return nil
    }
    return lastErr
}

func downloadImage(client *http.Client, urlStr, filename string) error {
    // 1. 创建带 3 秒超时的 context
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    // 2. Clone 预设请求模板，并注入带超时的 context
    req := baseDownloadReq.Clone(ctx)

    // 3. 设置 URL（每次不同）
    u, err := url.Parse(urlStr)
    if err != nil {
        return err
    }
    req.URL = u

    // 4. 发送请求（后续逻辑不变）
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("status %d", resp.StatusCode)
    }

    file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
    if err != nil {
        return err
    }
    defer file.Close()

    buffer := bufferPool.Get().([]byte)
    defer bufferPool.Put(buffer)

    _, err = io.CopyBuffer(file, resp.Body, buffer)
    return err
}
