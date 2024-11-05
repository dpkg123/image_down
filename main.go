package main

import (
    "fmt"
    "io"
    "net/http"
    "os"
    "sync"
    "time"
)

func main() {
    folder := "downloads"
    number := 1000// 下载图片数量

    if err := os.MkdirAll(folder, os.ModePerm); err != nil {
        fmt.Println("Failed to create directory:", err)
        return
    }

    url := "https://api.iw233.cn/api.php?sort=random"
    concurrentDownloads := 5
    semaphore := make(chan struct{}, concurrentDownloads)

    var wg sync.WaitGroup
    start := time.Now()

    for i := 1; i <= number; i++ {
        wg.Add(1)
        semaphore <- struct{}{}

        go func(i int) {
            defer wg.Done()
            defer func() { <-semaphore }()

            filename := fmt.Sprintf("%s/%d.jpg", folder, time.Now().UnixNano())
            if err := downloadWithRetry(url, filename, 3); err != nil {
                fmt.Printf("Failed to download image %d: %v\n", i, err)
            } else {
                fmt.Printf("Downloaded image %d to %s\n", i, filename)
            }
        }(i)
    }

    wg.Wait()
    fmt.Printf("Downloaded %d images to %s in %v\n", number, folder, time.Since(start))
}

func downloadWithRetry(url, filename string, retries int) error {
    for i := 0; i <= retries; i++ {
        if err := downloadFile(url, filename); err != nil {
            if i < retries {
                fmt.Printf("Retrying %s (%d/%d)\n", filename, i+1, retries)
                time.Sleep(2 * time.Second) // 等待 2 秒再重试
            } else {
                return err
            }
        } else {
            return nil
        }
    }
    return fmt.Errorf("failed to download %s after %d attempts", filename, retries)
}

func downloadFile(url, filename string) error {
    client := &http.Client{
        Timeout: 10 * time.Second,
    }
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return err
    }
    req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Go-http-client/1.1)")

    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("bad status: %s", resp.Status)
    }

    out, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer out.Close()

    _, err = io.Copy(out, resp.Body)
    return err
}
