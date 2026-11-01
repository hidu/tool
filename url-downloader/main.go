package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const outputDiscard = "-"

var (
	referer     = flag.String("referer", "", "Custom Referer")
	input       = flag.String("i", "urls.txt", "URL task file")
	output      = flag.String("o", "downloads", "Output directory ('-' to discard downloads)")
	concurrency = flag.Int("j", 1, "Number of concurrent downloads")
	userAgent   = flag.String("ua", "", "Custom User-Agent")
	timeout     = flag.Int("t", 600, "HTTP request timeout in seconds (0 = no timeout)")
)

func main() {
	flag.Parse()

	if *concurrency < 1 {
		fmt.Fprintln(os.Stderr, "Error: concurrency must be greater than 0")
		os.Exit(1)
	}

	if *timeout < 0 {
		fmt.Fprintln(os.Stderr, "Error: timeout cannot be negative")
		os.Exit(1)
	}

	if err := run(
		*input,
		*output,
		*concurrency,
		time.Duration(*timeout)*time.Second,
	); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

type Task struct {
	URL       string `json:"url"`
	Referer   string `json:"referer"`
	UserAgent string `json:"ua"`
	Out       string `json:"out"`
	Err       error  `json:"-"`
	Duplicate bool   `json:"-"`
}

func (t Task) isDiscard() bool {
	return t.Out == outputDiscard || *output == outputDiscard
}

// outputPath 完整的输出文件地址
func (t Task) outputPath() string {
	return filepath.Join(*output, t.Out)
}

type Result int

const (
	ResultSuccess Result = iota
	ResultSkipped
	ResultFailed
)

func run(inputFile string, outputDir string, concurrency int, timeout time.Duration) error {
	tasks, err := loadTasks(inputFile)
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return nil
	}

	if outputDir != outputDiscard {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return err
		}
	}

	client := &http.Client{
		Timeout: timeout,
	}

	total := len(tasks)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		success int
		skipped int
		failed  int
		done    int
	)

	jobs := make(chan Task)

	// Start workers.
	for range concurrency {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for task := range jobs {
				var (
					result Result
					err    error
				)
				start := time.Now()
				switch {
				case task.Err != nil:
					result = ResultFailed
					err = task.Err

				case task.Duplicate:
					result = ResultSkipped

				default:
					result, err = downloadTask(client, task)
				}
				cost := time.Since(start)

				mu.Lock()

				done++

				switch result {
				case ResultSuccess:
					success++
					fmt.Printf("[%d/%d] [OK]     %s  %d ms\n",
						done, total, filepath.ToSlash(task.Out), cost.Milliseconds())

				case ResultSkipped:
					skipped++
					fmt.Printf("[%d/%d] [SKIP]   %s (already exists or duplicate)\n",
						done, total, filepath.ToSlash(task.Out))

				case ResultFailed:
					failed++
					fmt.Printf("[%d/%d] [FAILED] %s: %v\n",
						done, total, task.URL, err)
				}

				mu.Unlock()
			}
		}()
	}

	for _, task := range tasks {
		jobs <- task
	}

	close(jobs)
	wg.Wait()

	fmt.Println()
	fmt.Printf(
		"Total: %d | Success: %d | Skipped: %d | Failed: %d\n",
		total, success, skipped, failed,
	)

	if failed > 0 {
		return fmt.Errorf("%d task(s) failed", failed)
	}

	return nil
}

// loadTasks reads and parses the three supported task formats.
func loadTasks(fileName string) ([]Task, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var tasks []Task

	// Prevent duplicate output paths.
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for line := 0; scanner.Scan(); line++ {
		raw := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments.
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}

		task, err := parseTask(raw)
		if err != nil {
			tasks = append(tasks, Task{
				URL: raw,
				Err: fmt.Errorf("line %d: %w", line+1, err),
			})
			continue
		}

		// Normalize output path for duplicate detection.
		key := strings.ToLower(filepath.Clean(task.Out))

		if seen[key] {
			task.Duplicate = true
		} else {
			seen[key] = true
		}

		if task.Referer == "" {
			task.Referer = *referer
		}

		if task.UserAgent == "" {
			task.UserAgent = *userAgent
		}

		tasks = append(tasks, task)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

// parseTask parses a single task line.
//
// Supported formats:
// 1. URL
// 2. URL output_path
// 3. JSON: {"url":"...", "out":"..."}
func parseTask(raw string) (Task, error) {
	var task Task

	switch {
	case strings.HasPrefix(raw, "{"):
		// JSON format.
		if err := json.Unmarshal([]byte(raw), &task); err != nil {
			return task, fmt.Errorf("invalid JSON: %w", err)
		}

	case strings.HasPrefix(raw, "http://"),
		strings.HasPrefix(raw, "https://"):
		// URL or URL + output path.
		fields := strings.Fields(raw)

		if len(fields) < 1 || len(fields) > 2 {
			return task, errors.New("expected URL or URL + output path")
		}

		task.URL = fields[0]

		if len(fields) == 2 {
			task.Out = fields[1]
		}

	default:
		return task, errors.New("unsupported task format")
	}

	// Validate URL.
	u, err := url.Parse(task.URL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return task, errors.New("invalid HTTP/HTTPS URL")
	}

	if *output == outputDiscard {
		task.Out = outputDiscard
	}

	// Automatically extract filename if output path is omitted.
	if task.Out == "" {
		fileName := strings.TrimLeft(u.Path, "/")
		if fileName == "" {
			fileName = "index.html"
		}
		if fileName == "." {
			return task, errors.New("cannot extract filename from URL")
		}

		task.Out, err = url.PathUnescape(fileName)
		if err != nil {
			return task, errors.New("invalid URL filename")
		}
	}

	if !task.isDiscard() {
		// Validate output path.
		task.Out = filepath.Clean(filepath.FromSlash(task.Out))

		if task.Out == "." ||
			filepath.IsAbs(task.Out) ||
			task.Out == ".." ||
			strings.HasPrefix(task.Out, ".."+string(filepath.Separator)) {
			return task, errors.New("invalid output path: must be a relative path")
		}
	}

	return task, nil
}

// downloadTask downloads a single file.
func downloadTask(client *http.Client, task Task) (Result, error) {
	if !task.isDiscard() {
		// Skip existing files.
		info, err := os.Stat(task.outputPath())

		if err == nil {
			if !info.IsDir() {
				return ResultSkipped, nil
			}
			return ResultFailed, fmt.Errorf("output path %q is a directory", task.outputPath())
		}

		if !os.IsNotExist(err) {
			return ResultFailed, err
		}
	}
	if err := downloadFile(client, task); err != nil {
		return ResultFailed, err
	}

	return ResultSuccess, nil
}

func setRequest(req *http.Request) {
	h := req.Header
	h.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	h.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	h.Set("Accept-Encoding", "gzip, deflate")
	h.Set("Cache-Control", "no-cache")
	h.Set("Pragma", "no-cache")
}

// downloadFile downloads to a temporary file before renaming.
func downloadFile(client *http.Client, task Task) error {
	req, err := http.NewRequest(http.MethodGet, task.URL, nil)
	if err != nil {
		return err
	}

	if task.UserAgent != "" {
		req.Header.Set("User-Agent", task.UserAgent)
	}
	if task.Referer != "" {
		req.Header.Set("Referer", task.Referer)
	}
	setRequest(req)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP status: %s", resp.Status)
	}

	if task.isDiscard() {
		_, err = io.Copy(io.Discard, resp.Body)
		return err
	}

	outputPath := task.outputPath()

	dir := filepath.Dir(outputPath)
	// Create parent directories for nested output paths.
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create a temporary file in the destination directory.
	tmp, err := os.CreateTemp(dir, ".download-*")
	if err != nil {
		return err
	}

	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	_, copyErr := io.Copy(tmp, resp.Body)
	closeErr := tmp.Close()

	if copyErr != nil {
		return copyErr
	}

	if closeErr != nil {
		return closeErr
	}

	// Check again before renaming.
	if _, err := os.Stat(outputPath); err == nil {
		return errors.New("file already exists")
	} else if !os.IsNotExist(err) {
		return err
	}

	return os.Rename(tmpPath, outputPath)
}
