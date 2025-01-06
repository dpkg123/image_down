package downloadimg

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Usage: ./delete_image_and_download <min_img_size> <max_img_size> <number>")
		os.Exit(1)
	}

	// 获取命令行参数
	minImgSize, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("Invalid min_img_size:", err)
		os.Exit(1)
	}

	maxImgSize, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println("Invalid max_img_size:", err)
		os.Exit(1)
	}

	number, err := strconv.Atoi(os.Args[3])
	if err != nil {
		fmt.Println("Invalid number of images to download:", err)
		os.Exit(1)
	}

	folder := "./images" // 设置默认文件夹位置

	// 处理图片删除
	for {
		// 删除小于 minImgSize 或大于 maxImgSize 的图片
		deleteImages(minImgSize, maxImgSize)

		// 计算当前图像数量
		lastImgCount := countImages()

		// 检查是否需要下载更多图片
		if lastImgCount < number {
			// 下载更多图片
			downloadImages(folder, number-lastImgCount)
		} else {
			// 满足需求，跳出循环
			break
		}
	}
}

func deleteImages(minSize, maxSize int) {
	// 删除小于 minSizeMB 或大于 maxSizeMB 的图片
	deleteCommand := fmt.Sprintf("find . -type f \\( -name \"*.jpeg\" -o -name \"*.png\" -o -name \"*.jpg\" \\) -size -%dM -print -delete", minSize)
	executeCommand(deleteCommand)

	deleteCommand = fmt.Sprintf("find . -type f \\( -name \"*.jpeg\" -o -name \"*.png\" -o -name \"*.jpg\" \\) -size +%dM -print -delete", maxSize)
	executeCommand(deleteCommand)
}

func countImages() int {
	findCommand := `find . -type f \( -name "*.jpeg" -o -name "*.png" -o -name "*.jpg" \)`
	cmd := exec.Command("sh", "-c", findCommand+" | wc -l")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error counting images:", err)
		os.Exit(1)
	}

	count, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		fmt.Println("Error parsing image count:", err)
		os.Exit(1)
	}
	return count
}

func downloadImages(folder string, number int) {
	// 确保目标文件夹存在
	if err := os.MkdirAll(folder, os.ModePerm); err != nil {
		fmt.Println("Error creating folder:", err)
		os.Exit(1)
	}

	// 定义待测试的站点列表
	urls := []string{
		"https://dev.iw233.cn/api.php?sort=random",
		"https://api.iw233.cn/api.php?sort=random",
		"https://iw233.cn/api.php?sort=random",
	}

	// 检测每个站点
	var fastestURL string
	minTime := float64(999999)

	for _, url := range urls {
		// 使用 http 测试连接时间
		timeTaken, err := testSite(url)
		if err != nil {
			fmt.Printf("Error testing site %s: %v\n", url, err)
			continue
		}

		if timeTaken < minTime {
			minTime = timeTaken
			fastestURL = url
		}
	}

	if fastestURL == "" {
		fmt.Println("No available site found. Exiting.")
		os.Exit(1)
	}

	fmt.Printf("Using the fastest site: %s\n", fastestURL)

	// 下载图片
	client := &http.Client{}
	concurrentDownloads := 25
	done := make(chan bool)

	for i := 0; i < number; i++ {
		go func(i int) {
			filename := fmt.Sprintf("%s/%d.jpg", folder, time.Now().UnixNano())

			req, err := http.NewRequest("GET", fastestURL, nil)
			if err != nil {
				fmt.Printf("Error creating request for image %d: %v\n", i, err)
				done <- false
				return
			}

			req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.2903.9")

			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("Error downloading image %d: %v\n", i, err)
				done <- false
				return
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error reading image data %d: %v\n", i, err)
				done <- false
				return
			}

			err = ioutil.WriteFile(filename, body, 0644)
			if err != nil {
				fmt.Printf("Error saving image %d: %v\n", i, err)
				done <- false
				return
			}

			fmt.Printf("Downloaded image %d to %s\n", i, filename)
			done <- true
		}(i)
	}

	// 等待所有下载完成
	for i := 0; i < number; i++ {
		<-done
	}
}

func testSite(url string) (float64, error) {
	start := time.Now()
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	duration := time.Since(start).Seconds()
	return duration, nil
}

func executeCommand(command string) {
	cmd := exec.Command("sh", "-c", command)
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error executing command:", err)
		os.Exit(1)
	}
}
