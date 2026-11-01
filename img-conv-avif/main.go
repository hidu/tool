package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const (
	colorRed   = "\033[31m"
	colorReset = "\033[0m"

	statusOK     = "OK"
	statusKeep   = "KEEP"
	statusCopy   = "COPY"
	statusSkip   = "SKIP"
	statusIgnore = "IGNORE"
	statusFailed = "FAILED"
)

type Stats struct {
	Total      int
	Success    int
	Skipped    int
	Preserved  int
	Copied     int
	Ignored    int
	Failed     int
	BeforeSize int64
	AfterSize  int64
}

type Image struct {
	Path      string
	Rel       string
	Supported bool
	Format    string
}

type Result struct {
	Image      Image
	Dst        string
	Status     string
	BeforeSize int64
	AfterSize  int64
	Err        error
}

// String formats the conversion summary.
func (s Stats) String() string {
	return fmt.Sprintf(
		"========== Conversion Summary ==========\n"+
			"Total files:        %d\n"+
			"Converted:          %d\n"+
			"Original kept:      %d\n"+
			"Copied:             %d\n"+
			"Skipped:            %d\n"+
			"Ignored:            %d\n"+
			"Failed:             %d\n"+
			"----------------------------------------\n"+
			"Original size:      %s\n"+
			"Output size:        %s\n"+
			"Space saved:        %s\n"+
			"Compression rate:   %.2f%%",
		s.Total,
		s.Success,
		s.Preserved,
		s.Copied,
		s.Skipped,
		s.Ignored,
		s.Failed,
		formatSize(s.BeforeSize),
		formatSize(s.AfterSize),
		formatSize(s.BeforeSize-s.AfterSize),
		compressionRate(s.BeforeSize, s.AfterSize),
	)
}

// formatSize converts bytes to KB.
func formatSize(size int64) string {
	return fmt.Sprintf("%.2f KB", float64(size)/1024)
}

// compressionRate calculates the compression rate.
func compressionRate(before, after int64) float64 {
	if before <= 0 {
		return 0
	}

	return (1 - float64(after)/float64(before)) * 100
}

// isSupportedImage checks whether the file extension is supported.
func isSupportedImage(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case
		".avif",
		".bmp", ".dib",
		".gif",
		".heic", ".heif",
		".ico", ".cur",
		".jfif", ".jpe", ".jpeg", ".jpg",
		".jxl",
		".jp2", ".j2k", ".jpf", ".jpx", ".jpm", ".mj2",
		".png", ".apng",
		".psd", ".psb",
		// ".svg",
		".tif", ".tiff",
		".webp",
		".tga",
		".pcx",
		".dds",
		".exr", ".hdr",
		".jng", ".mng",
		".pbm", ".pgm", ".ppm", ".pnm", ".pfm", ".pam",
		".pict", ".pic",
		".ras",
		".sgi", ".sun",
		".wbmp",
		".xbm", ".xpm":
		return true
	}

	return false
}

// scanImages recursively discovers files.
func scanImages(srcDir string) ([]Image, error) {
	var images []Image

	err := filepath.WalkDir(srcDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			fmt.Fprintf(
				os.Stderr,
				"%s[SCAN ERROR] %s: %v%s\n",
				colorRed, path, walkErr, colorReset,
			)
			return nil
		}

		if d.IsDir() || !d.Type().IsRegular() {
			return nil
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		images = append(images, Image{
			Path:      path,
			Rel:       rel,
			Supported: isSupportedImage(path),
			Format:    strings.TrimPrefix(strings.ToUpper(filepath.Ext(path)), "."),
		})

		return nil
	})

	return images, err
}

// copyFileAtomic copies a file using a temporary file.
func copyFileAtomic(src, dst string) error {
	temp, err := os.CreateTemp(filepath.Dir(dst), ".copy-*")
	if err != nil {
		return err
	}

	tempPath := temp.Name()
	defer os.Remove(tempPath)

	input, err := os.Open(src)
	if err != nil {
		temp.Close()
		return err
	}
	defer input.Close()

	if _, err := io.Copy(temp, input); err != nil {
		temp.Close()
		return err
	}

	if err := temp.Close(); err != nil {
		return err
	}

	return os.Rename(tempPath, dst)
}

// outputPath returns the destination path for an image.
func outputPath(img Image, dstDir string) string {
	if img.Supported {
		name := strings.TrimSuffix(img.Rel, filepath.Ext(img.Rel)) + ".avif"
		return filepath.Join(dstDir, name)
	}

	return filepath.Join(dstDir, img.Rel)
}

// convertImage converts an image to AVIF.
func convertImage(img Image, dst string, quality int) (string, int64, int64, error) {
	if _, err := os.Stat(dst); err == nil {
		return statusSkip, 0, 0, nil
	} else if !os.IsNotExist(err) {
		return statusFailed, 0, 0, err
	}

	sourceInfo, err := os.Stat(img.Path)
	if err != nil {
		return statusFailed, 0, 0, err
	}

	beforeSize := sourceInfo.Size()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return statusFailed, 0, 0, err
	}

	temp, err := os.CreateTemp(filepath.Dir(dst), ".avif-*.avif")
	if err != nil {
		return statusFailed, 0, 0, err
	}

	tempPath := temp.Name()
	if err := temp.Close(); err != nil {
		os.Remove(tempPath)
		return statusFailed, 0, 0, err
	}
	defer os.Remove(tempPath)

	cmd := exec.Command(
		"magick",
		img.Path,
		"-quality", strconv.Itoa(quality),
		tempPath,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return statusFailed, 0, 0, fmt.Errorf(
			"%w: %s",
			err,
			strings.TrimSpace(string(output)),
		)
	}

	outputInfo, err := os.Stat(tempPath)
	if err != nil {
		return statusFailed, 0, 0, err
	}

	afterSize := outputInfo.Size()

	// Preserve the original AVIF if conversion increases file size.
	if strings.EqualFold(img.Format, "AVIF") && afterSize > beforeSize {
		if err := copyFileAtomic(img.Path, dst); err != nil {
			return statusFailed, 0, 0, err
		}

		return statusKeep, beforeSize, beforeSize, nil
	}

	if err := os.Rename(tempPath, dst); err != nil {
		return statusFailed, 0, 0, err
	}

	return statusOK, beforeSize, afterSize, nil
}

// copyUnknownFile copies an unsupported file without modification.
func copyUnknownFile(img Image, dst string) (string, int64, int64, error) {
	if _, err := os.Stat(dst); err == nil {
		return statusSkip, 0, 0, nil
	} else if !os.IsNotExist(err) {
		return statusFailed, 0, 0, err
	}

	sourceInfo, err := os.Stat(img.Path)
	if err != nil {
		return statusFailed, 0, 0, err
	}

	beforeSize := sourceInfo.Size()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return statusFailed, 0, 0, err
	}

	if err := copyFileAtomic(img.Path, dst); err != nil {
		return statusFailed, 0, 0, err
	}

	return statusCopy, beforeSize, beforeSize, nil
}

// worker processes files concurrently.
func worker(
	jobs <-chan Image,
	results chan<- Result,
	dstDir string,
	quality int,
	copyUnknown bool,
) {
	for img := range jobs {
		dst := outputPath(img, dstDir)

		var status string
		var before, after int64
		var err error

		switch {
		case img.Supported:
			status, before, after, err = convertImage(img, dst, quality)

		case copyUnknown:
			status, before, after, err = copyUnknownFile(img, dst)

		default:
			status = statusIgnore
		}

		results <- Result{
			Image:      img,
			Dst:        dst,
			Status:     status,
			BeforeSize: before,
			AfterSize:  after,
			Err:        err,
		}
	}
}

// printResult prints the processing result.
func printResult(index, total int, result Result) {
	prefix := fmt.Sprintf("[%d/%d]", index, total)

	switch result.Status {
	case statusOK:
		fmt.Printf(
			"%s [OK] %s -> %s | %s -> %s | Compression: %.2f%%\n",
			prefix,
			result.Image.Rel,
			filepath.Base(result.Dst),
			formatSize(result.BeforeSize),
			formatSize(result.AfterSize),
			compressionRate(result.BeforeSize, result.AfterSize),
		)

	case statusKeep:
		fmt.Printf(
			"%s [KEEP] %s -> %s | AVIF output was larger; original preserved | %s\n",
			prefix,
			result.Image.Rel,
			filepath.Base(result.Dst),
			formatSize(result.BeforeSize),
		)

	case statusCopy:
		fmt.Printf(
			"%s [COPY] %s -> %s | Unsupported format; copied unchanged | %s\n",
			prefix,
			result.Image.Rel,
			filepath.Base(result.Dst),
			formatSize(result.BeforeSize),
		)

	case statusSkip:
		fmt.Printf(
			"%s [SKIP] %s -> %s | Destination already exists\n",
			prefix,
			result.Image.Rel,
			filepath.Base(result.Dst),
		)

	case statusIgnore:
		fmt.Printf(
			"%s [IGNORE] %s -> %s | Unsupported format; not copied\n",
			prefix,
			result.Image.Rel,
			filepath.Base(result.Dst),
		)

	case statusFailed:
		fmt.Printf(
			"%s%s [FAILED] %s -> %s | %v%s\n",
			colorRed,
			prefix,
			result.Image.Rel,
			filepath.Base(result.Dst),
			result.Err,
			colorReset,
		)
	}
}

func main() {
	inputDir := flag.String("i", "", "Input directory")
	quality := flag.Int("q", 60, "AVIF quality (1-100)")
	concurrency := flag.Int("j", 1, "Number of concurrent workers")
	copyUnknown := flag.Bool(
		"copy-unsupported",
		true,
		"Copy unsupported files without modification",
	)

	flag.Parse()

	if strings.TrimSpace(*inputDir) == "" ||
		filepath.Clean(*inputDir) == "." {
		fmt.Fprintf(
			os.Stderr,
			"%sError: -i must specify a valid directory.%s\n",
			colorRed, colorReset,
		)
		flag.Usage()
		os.Exit(1)
	}

	if *quality < 1 || *quality > 100 {
		fmt.Fprintf(
			os.Stderr,
			"%sError: -q must be between 1 and 100.%s\n",
			colorRed, colorReset,
		)
		os.Exit(1)
	}

	if *concurrency < 1 {
		fmt.Fprintf(
			os.Stderr,
			"%sError: -j must be at least 1.%s\n",
			colorRed, colorReset,
		)
		os.Exit(1)
	}

	srcDir, err := filepath.Abs(*inputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError: %v%s\n",
			colorRed, err, colorReset)
		os.Exit(1)
	}

	info, err := os.Stat(srcDir)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(
			os.Stderr,
			"%sError: Input directory does not exist.%s\n",
			colorRed, colorReset,
		)
		os.Exit(1)
	}

	if _, err := exec.LookPath("magick"); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"%sError: ImageMagick was not found in PATH.%s\n",
			colorRed, colorReset,
		)
		os.Exit(1)
	}

	dstDir := filepath.Clean(srcDir) + "-avif"

	fmt.Printf("Input directory:  %s\n", srcDir)
	fmt.Printf("Output directory: %s\n", dstDir)
	fmt.Printf("AVIF quality:     %d\n", *quality)
	fmt.Printf("Concurrency:      %d\n", *concurrency)
	fmt.Printf("Copy unknown:     %t\n\n", *copyUnknown)

	fmt.Println("Scanning files...")

	images, err := scanImages(srcDir)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"%sError: Directory scanning failed: %v%s\n",
			colorRed, err, colorReset,
		)
		os.Exit(1)
	}

	total := len(images)
	fmt.Printf("Found %d files.\n\n", total)

	if total == 0 {
		return
	}

	workers := *concurrency
	if workers > total {
		workers = total
	}

	jobs := make(chan Image, workers)
	results := make(chan Result, workers)

	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			worker(jobs, results, dstDir, *quality, *copyUnknown)
		}()
	}

	go func() {
		for _, img := range images {
			jobs <- img
		}

		close(jobs)
		wg.Wait()
		close(results)
	}()

	var stats Stats
	stats.Total = total

	for result := range results {
		completed := stats.Success +
			stats.Skipped +
			stats.Preserved +
			stats.Copied +
			stats.Ignored +
			stats.Failed + 1

		printResult(completed, total, result)

		switch result.Status {
		case statusOK:
			stats.Success++
			stats.BeforeSize += result.BeforeSize
			stats.AfterSize += result.AfterSize

		case statusKeep:
			stats.Preserved++
			stats.BeforeSize += result.BeforeSize
			stats.AfterSize += result.AfterSize

		case statusCopy:
			stats.Copied++
			stats.BeforeSize += result.BeforeSize
			stats.AfterSize += result.AfterSize

		case statusSkip:
			stats.Skipped++

		case statusIgnore:
			stats.Ignored++

		case statusFailed:
			stats.Failed++
		}
	}

	fmt.Println()
	fmt.Println(stats)
	fmt.Printf("Output directory:   %s\n", dstDir)
}
