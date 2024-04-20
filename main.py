import requests
from tqdm import tqdm                                                                           import hashlib                                                                                  import threading
import argparse
from time import time as timer

image_hashes = []
host_urls = [
    "https://dev.iw233.cn/api.php?sort=random",
    "https://api.iw233.cn/api.php?sort=random",
    "https://iw233.cn/api.php?sort=random"                                                      ]                                                                                                                                                                                               def download_image(url, file_path):
    response = requests.get(url)                                                                    img = response.content                                                                          md5_hash = hashlib.md5(img).hexdigest()                                                     
    if md5_hash not in image_hashes:
        image_hashes.append(md5_hash)
        with open(file_path, "wb") as f:                                                                    f.write(img)
        print("Saved image to: ", file_path)
    else:
        print("Duplicate image found!")                                                                                                                                                         def main(images_count, threads_count):                                                              start = timer()                                                                                 threads = [threading.Thread(target=download_image,
                                args=(host_urls[i % len(host_urls)],
                                      f"img/{i}.jpg"))                                                         for i in range(images_count)]                                                    
    for i in range(0, images_count, threads_count):
        start_idx = i
        end_idx = min(i + threads_count, images_count)
        print(f"Starting download for images: {start_idx} to {end_idx}")

        threads_slice = threads[start_idx:end_idx]                                              
        for thread in threads_slice:                                                                        thread.start()                                                                                                                                                                              for thread in threads_slice:
            thread.join()
                                                                                                        print(f"Finished download for images: {start_idx} to {end_idx}")                        
    print(f"Time taken: {timer() - start}")                                                     
if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("-n", "--number", help="Number of images to download", type=int, required=True)
    parser.add_argument("-t", "--threads", help="Number of threads to use", type=int, default=1)
    args = parser.parse_args()                                                                                                                                                                      main(args.number, args.threads)
