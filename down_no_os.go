package noos

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Usage: ./delete_image_and_download <min_img_size> <max_img_size> <number>")
		return
	}

	// 获取命令行参数
	minImgSize, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("Invalid min_img_size:", err)
		return
	}

	maxImgSize, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println("Invalid max_img_size:", err)
		return
	}

	number, err := strconv.Atoi(os.Args[3])
	if err != nil {
		fmt.Println("Invalid number of images to download:", err)
		return
	}

	folder := "./images" // 设置默认文件夹位置

	// 处理图片删除
	for {
		// 删除小于 minImgSize 或大于 maxImgSize 的图片
		deleteImages(minImgSize, maxSize)

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
	// 遍历当前目录中的所有文件
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 检查是否为图片文件
		if strings.HasSuffix(info.Name(), ".jpeg") || strings.HasSuffix(info.Name(), ".png") || strings.HasSuffix(info.Name(), ".jpg") {
			// 获取文件大小
			sizeMB := info.Size() / (1024 * 1024) // 转换为 MB

			// 如果文件大小不符合条件，则删除
			if sizeMB < minSize || sizeMB > maxSize {
				err := deleteFile(path)
				if err != nil {
					fmt.Printf("Failed to delete %s: %v\n", path, err)
				} else {
					fmt.Printf("Deleted %s\n", path)
				}
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error walking through files:", err)
	}
}

func countImages() int {
	count := 0
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 检查是否为图片文件
		if strings.HasSuffix(info.Name(), ".jpeg") || strings.HasSuffix(info.Name(), ".png") || strings.HasSuffix(info.Name(), ".jpg") {
			count++
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error counting images:", err)
	}
	return count
}

func downloadImages(folder string, number int) {
	// 确保目标文件夹存在
	err := createFolder(folder)
	if err != nil {
		fmt.Println("Error creating folder:", err)
		return
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
		return
	}

	fmt.Printf("Using the fastest site: %s\n", fastestURL)

	// 下载图片
	var wg sync.WaitGroup
	client := &http.Client{}

	for i := 0; i < number; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			filename := fmt.Sprintf("%s/%d.jpg", folder, time.Now().UnixNano())

			req, err := http.NewRequest("GET", fastestURL, nil)
			if err != nil {
				fmt.Printf("Error creating request for image %d: %v\n", i, err)
				return
			}

			req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.2903.9")

			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("Error downloading image %d: %v\n", i, err)
				return
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error reading image data %d: %v\n", i, err)
				return
			}

			err = ioutil.WriteFile(filename, body, 0644)
			if err != nil {
				fmt.Printf("Error saving image %d: %v\n", i, err)
				return
			}

			fmt.Printf("Downloaded image %d to %s\n", i, filename)
		}(i)
	}

	// 等待所有下载完成
	wg.Wait()
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

func deleteFile(path string) error {
	err := os.Remove(path)
	return err
}

func createFolder(folder string) error {
	err := os.MkdirAll(folder, 0755)
	return err
}
