defmodule Downloader do
  def download_images(total) do
    # 模拟 UA
    ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"

    # 确保引入了 HTTPoison 模块
    {:ok, _} = HTTPoison.start()

    # 并行下载
    1..total
    |> Task.async_stream(
      fn i ->
        filename = "image-2025-01-11/#{:os.system_time(:seconds) * 1_000_000_000}.jpg"
        response = HTTPoison.get!("https://api.iw233.cn/api.php?sort=random", [
          {"User-Agent", ua},
          {"Referer", "https://weibo.com/"},
          {"Accept-Language", "zh-CN,cn;q=0.9"}
        ])

        case response do
          {:ok, %{status_code: 200, body: body}} ->
            File.write!(filename, body)
            IO.puts("Downloaded #{i} of #{total}. filename: #{filename}")
          _ ->
            IO.puts("Failed to download image #{i}")
        end
      end,
      max_concurrency: 50
    )
    |> Stream.run()
  end
end

