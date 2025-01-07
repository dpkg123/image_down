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
    concurrentDownloads = 25
    userAgent          = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.2903.9"
)

var urls = []string{
    "https://dev.iw233.cn/api.php?sort=random",
    "https://api.iw233.cn/api.php?sort=random",
    "https://iw233.cn/api.php?sort=random",
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

    // Create HTTP client with custom settings
    client := &http.Client{
        Timeout: 30 * time.Second,
    }

    // Create semaphore for limiting concurrent downloads
    sem := make(chan struct{}, concurrentDownloads)
    var wg sync.WaitGroup

    // Download images
    for i := 1; i <= number; i++ {
        wg.Add(1)
        sem <- struct{}{} // Acquire semaphore

        go func(i int) {
            defer wg.Done()
            defer func() { <-sem }() // Release semaphore

            filename := filepath.Join(folder, fmt.Sprintf("%d.jpg", time.Now().UnixNano()))
            if err := downloadImage(client, fastestURL, filename); err != nil {
                fmt.Printf("Failed to download image %d: %v\n", i, err)
                return
            }
            fmt.Printf("Downloaded %d of %d. filename: %s\n", i, number, filename)
        }(i)
    }

    wg.Wait()

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

    _, err = io.Copy(file, resp.Body)
    return err
}
