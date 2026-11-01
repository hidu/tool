# url-downloader


## 1. TaskFile Example urls.txt
```text
# Format 1: URL only
https://example.com/files/aaa.jpg

# Format 2: URL + output path
https://example.com/files/bbb.jpg dir/bbb.jpg
https://example.com/files/ccc.jpg images/ccc.jpg

# Format 3: JSON
{"url":"https://example.com/files/ddd.jpg","out":"dir/ddd.jpg"}
{"url":"https://example.com/files/eee.jpg","out":"images/eee.jpg"}
```

## Usage
```bash
$ url-downloader -help
Usage of url-downloader:
  -i string
        URL task file (default "urls.txt")
  -j int
        Number of concurrent downloads (default 1)
  -o string
        Output directory (default "downloads")
  -referer string
        Custom Referer
  -t int
        HTTP request timeout in seconds (0 = no timeout) (default 600)
  -ua string
        Custom User-Agent
```
