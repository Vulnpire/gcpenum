package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Object struct {
	Name string `json:"name"`
}

type ObjectListResponse struct {
	Items []Object `json:"items"`
}

const (
	wordlistURL      = "https://raw.githubusercontent.com/Vulnpire/gcpenum/refs/heads/main/utils/wordlist.txt"
	wordlistFilename = ".config/gcpenum/words.txt"
)

func ensureWordlist() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("ERROR: Unable to locate home directory: %v\n", err)
		os.Exit(1)
	}

	wordlistPath := filepath.Join(homeDir, wordlistFilename)
	if _, err := os.Stat(wordlistPath); os.IsNotExist(err) {
		fmt.Printf("Wordlist not found. Downloading to %s...\n", wordlistPath)
		dir := filepath.Dir(wordlistPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("ERROR: Could not create directory %s: %v\n", dir, err)
			os.Exit(1)
		}
		if err := downloadFile(wordlistURL, wordlistPath); err != nil {
			fmt.Printf("ERROR: Failed to download wordlist: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("Using existing wordlist at %s\n", wordlistPath)
	}
	return wordlistPath
}

func downloadFile(url, filePath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s (status: %d)", url, resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	return ioutil.WriteFile(filePath, data, 0644)
}

func generatePermutations(keyword string, wordlistPath string) []string {
	permutations := []string{
		"{keyword}-{suffix}",
		"{suffix}-{keyword}",
		"{keyword}_{suffix}",
		"{suffix}_{keyword}",
		"{keyword}{suffix}",
		"{suffix}{keyword}",
	}

	suffixes := readLinesFromFile(wordlistPath)
	var buckets []string

	for _, suffix := range suffixes {
		for _, template := range permutations {
			bucket := strings.ReplaceAll(template, "{keyword}", keyword)
			bucket = strings.ReplaceAll(bucket, "{suffix}", suffix)
			buckets = append(buckets, bucket)
		}
	}

	buckets = append(buckets, keyword, keyword+".com", keyword+".net", keyword+".org")
	return removeDuplicates(buckets)
}

func removeDuplicates(input []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, v := range input {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

func readLinesFromFile(filePath string) []string {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("ERROR: Unable to read file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("ERROR: Unable to read file: %v\n", err)
		os.Exit(1)
	}
	return lines
}

func checkBucket(bucket string, verbose bool, wg *sync.WaitGroup, output chan string) {
	defer wg.Done()

	bucketURL := fmt.Sprintf("https://storage.googleapis.com/%s/", bucket)
	apiURL := fmt.Sprintf("https://www.googleapis.com/storage/v1/b/%s", bucket)

	resp, err := http.Head(apiURL)
	if err != nil {
		output <- fmt.Sprintf("ERROR: Could not connect to %s - %v", apiURL, err)
		return
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 404:
		return
	case 403:
		body, _ := ioutil.ReadAll(resp.Body)
		if bytes.Contains(body, []byte("Access denied")) || bytes.Contains(body, []byte("does not have")) {
			return
		}
		output <- fmt.Sprintf("EXISTS: %s", bucketURL)
	case 200:
		output <- fmt.Sprintf("EXISTS: %s", bucketURL)
		listObjects(bucket, output)
	default:
		if verbose {
			output <- fmt.Sprintf("UNKNOWN RESPONSE for %s: %d", bucketURL, resp.StatusCode)
		}
	}
}

func listObjects(bucket string, output chan string) {
	apiURL := fmt.Sprintf("https://www.googleapis.com/storage/v1/b/%s/o", bucket)

	resp, err := http.Get(apiURL)
	if err != nil {
		output <- fmt.Sprintf("ERROR: Could not list objects in %s - %v", bucket, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var objectList ObjectListResponse
		if err := json.NewDecoder(resp.Body).Decode(&objectList); err != nil {
			output <- fmt.Sprintf("ERROR: Could not parse object list for %s - %v", bucket, err)
			return
		}
		output <- fmt.Sprintf("    LISTABLE: %s", bucket)
		for _, obj := range objectList.Items {
			output <- fmt.Sprintf("        - %s", obj.Name)
		}
	}
}

func main() {
    keyword := flag.String("n", "", "Keyword for bucket name permutations")
    wordlist := flag.String("w", "", "Path to a wordlist file (defaults to downloaded wordlist)")
    outFile := flag.String("o", "", "Path to save the results")
    subprocesses := flag.Int("c", 10, "Number of concurrent processes")
    verbose := flag.Bool("v", false, "Enable verbose mode for detailed responses")
    keywordList := flag.String("l", "", "Path to a file containing a list of keywords")
    flag.Parse()

    if *keyword == "" && *keywordList == "" {
        fmt.Println("ERROR: Provide either a keyword (-n) or a keyword list file (-l)")
        flag.Usage()
        return
    }

    wordlistPath := *wordlist
    if wordlistPath == "" {
        wordlistPath = ensureWordlist()
    }

    var keywords []string
    if *keywordList != "" {
        keywords = readLinesFromFile(*keywordList)
    } else if *keyword != "" {
        keywords = []string{*keyword}
    }

    var buckets []string
    for _, kw := range keywords {
        buckets = append(buckets, generatePermutations(kw, wordlistPath)...)
    }

    fmt.Printf("\nGenerated %d bucket names from %d keyword(s).\n", len(buckets), len(keywords))

    var outputFile *os.File
    if *outFile != "" {
        var err error
        outputFile, err = os.Create(*outFile)
        if err != nil {
            fmt.Printf("ERROR: Could not create output file: %v\n", err)
            return
        }
        defer outputFile.Close()
    }

    output := make(chan string)
    var wg sync.WaitGroup

    startTime := time.Now()
    sem := make(chan struct{}, *subprocesses)

    for _, bucket := range buckets {
        wg.Add(1)
        go func(bucket string) {
            sem <- struct{}{}
            checkBucket(bucket, *verbose, &wg, output)
            <-sem
        }(bucket)
    }

    go func() {
        for line := range output {
            fmt.Println(line)
            if outputFile != nil {
                outputFile.WriteString(line + "\n")
            }
        }
    }()

    wg.Wait()
    close(output)

    duration := time.Since(startTime)
    fmt.Printf("\nScan completed in %s. Scanned %d buckets.\n", duration, len(buckets))
}
