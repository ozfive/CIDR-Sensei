package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	defaultConcurrency = 100
	defaultAlgorithm   = "binary-search"
	helpUsage          = "CIDR-Sensei -cidr=\"10.0.0.0/8,172.16.0.0/12,192.168.0.0/16\" -concurrency=100 -output json"
)

type CIDRRange struct {
	ipNet  *net.IPNet
	start  uint32
	end    uint32
	length uint32
}

type Config struct {
	OutputFormat string
	CIDRListStr  string
	Parallel     bool
	Concurrency  int
	Algorithm    string
}

func main() {
	// Parse flags and handle configuration
	config, err := parseFlags()
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}

	// Handle OS interrupts
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Parse CIDR list
	cidrRanges, err := parseCIDRList(strings.Split(config.CIDRListStr, ","))
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	// Start processing
	startTime := time.Now()

	var ips []string
	if config.Parallel {
		ips, err = cidrToIPsParallel(ctx, cidrRanges, config.Concurrency, config.Algorithm)
	} else {
		ips, err = cidrToIPsBinarySearch(cidrRanges)
	}

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	// Handle output
	err = handleOutput(config.OutputFormat, ips, config.CIDRListStr)
	if err != nil {
		fmt.Printf("Error writing output: %v\n", err)
	}

	fmt.Printf("Took %.2f seconds to complete.\n", time.Since(startTime).Seconds())
}

func parseFlags() (Config, error) {
	var config Config
	flag.StringVar(&config.OutputFormat, "output", "terminal", "the output format (json, csv, or terminal)")
	flag.StringVar(&config.CIDRListStr, "cidr", "", "a comma-separated list of CIDR blocks to expand into IPs")
	flag.BoolVar(&config.Parallel, "parallel", false, "enable parallel processing")
	flag.IntVar(&config.Concurrency, "concurrency", defaultConcurrency, "set the number of workers for parallel processing")
	flag.StringVar(&config.Algorithm, "algorithm", defaultAlgorithm, "the algorithm to use for expanding CIDR blocks into IPs (binary-search, interval-tree)")
	flag.Usage = func() {
		fmt.Printf("Usage: %s [OPTIONS]\n", os.Args[0])
		fmt.Println("Expand a comma-separated list of CIDR blocks into a list of IPs")
		fmt.Println("")
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println(helpUsage)
	}
	flag.Parse()

	// Validate flags
	if config.CIDRListStr == "" {
		return config, fmt.Errorf("the -cidr flag is required")
	}

	if config.Concurrency <= 0 {
		config.Concurrency = defaultConcurrency
	}

	if config.Algorithm != "binary-search" && config.Algorithm != "interval-tree" {
		config.Algorithm = defaultAlgorithm
	}

	return config, nil
}

func cidrToIPsParallel(ctx context.Context, cidrRanges []CIDRRange, concurrency int, algorithm string) ([]string, error) {
	var ips []string
	ipChan := make(chan string, 1000)
	errChan := make(chan error, 1)
	var wg sync.WaitGroup

	// Define the processing function based on the algorithm
	var processFunc func(CIDRRange, chan<- string, chan<- error)
	switch algorithm {
	case "interval-tree":
		tree := buildIntervalTree(cidrRanges)
		processFunc = func(cidr CIDRRange, out chan<- string, errs chan<- error) {
			for ip := cidr.start; ip <= cidr.end; ip++ {
				select {
				case <-ctx.Done():
					return
				default:
					if c := tree.search(ip); c != nil {
						out <- uint2ip(ip).String()
					}
				}
			}
		}
	case "binary-search":
		sort.Slice(cidrRanges, func(i, j int) bool { return cidrRanges[i].start < cidrRanges[j].start })
		processFunc = func(cidr CIDRRange, out chan<- string, errs chan<- error) {
			_ = errs // Explicitly ignore the errs parameter
			for ip := cidr.start; ip <= cidr.end; ip++ {
				select {
				case <-ctx.Done():
					return
				default:
					idx := sort.Search(len(cidrRanges), func(j int) bool {
						return cidrRanges[j].end >= ip
					})
					if idx < len(cidrRanges) && cidrRanges[idx].start <= ip {
						out <- uint2ip(ip).String()
					}
				}
			}
		}
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", algorithm)
	}

	// Start worker goroutines
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, cidr := range cidrRanges {
				select {
				case <-ctx.Done():
					return
				default:
					processFunc(cidr, ipChan, errChan)
				}
			}
		}()
	}

	// Close channels once all workers are done
	go func() {
		wg.Wait()
		close(ipChan)
		close(errChan)
	}()

	// Collect IPs from the channel
	for ip := range ipChan {
		ips = append(ips, ip)
	}

	// Check for errors
	if err, ok := <-errChan; ok {
		return nil, err
	}

	return ips, nil
}

func cidrToIPsBinarySearch(cidrRanges []CIDRRange) ([]string, error) {
	// Your existing sequential binary search implementation
	// Ensure it's optimized and remains after refactoring
	var ips []string

	// Sort the CIDR ranges by their start IP
	sortedCIDRRanges := make([]CIDRRange, len(cidrRanges))
	copy(sortedCIDRRanges, cidrRanges)
	sort.Slice(sortedCIDRRanges, func(i, j int) bool {
		return sortedCIDRRanges[i].start < sortedCIDRRanges[j].start
	})

	// Expand the CIDR ranges into a list of IPs using binary search
	for _, cidrRange := range sortedCIDRRanges {
		for i := cidrRange.start; i <= cidrRange.end; i++ {
			ip := uint2ip(i)
			idx := sort.Search(len(sortedCIDRRanges), func(j int) bool {
				return sortedCIDRRanges[j].end >= i
			})
			if idx < len(sortedCIDRRanges) && sortedCIDRRanges[idx].start <= i {
				ips = append(ips, ip.String())
			}
		}
	}

	return ips, nil
}

func buildIntervalTree(cidrRanges []CIDRRange) *intervalTree {
	tree := &intervalTree{}
	for _, cidr := range cidrRanges {
		err := tree.insert(cidr.start, cidr.end, &cidr)
		if err != nil {
			log.Fatalf("failed to insert CIDR range into interval tree: %v", err)
		}
	}
	return tree
}

func parseCIDRList(cidrList []string) ([]CIDRRange, error) {
	var cidrRanges []CIDRRange
	for _, cidrStr := range cidrList {
		ip, ipNet, err := net.ParseCIDR(cidrStr)
		if err != nil {
			return nil, fmt.Errorf("error parsing CIDR %s: %w", cidrStr, err)
		}
		start := ipToUint(ip)
		mask := ipNet.Mask
		// Calculate the end IP based on the mask
		end := start | ^ipToUint(net.IP(mask))
		cidrRanges = append(cidrRanges, CIDRRange{
			ipNet:  ipNet,
			start:  start,
			end:    end,
			length: end - start + 1,
		})
	}
	return cidrRanges, nil
}

func ipToUint(ip net.IP) uint32 {
	ipv4 := ip.To4()
	if ipv4 == nil {
		return 0 // Handle IPv6 or invalid IPs appropriately
	}
	return binary.BigEndian.Uint32(ipv4)
}

func uint2ip(ip uint32) net.IP {
	result := make(net.IP, 4)
	binary.BigEndian.PutUint32(result, ip)
	return result
}

func (t *intervalTree) insert(start, end uint32, cidr *CIDRRange) error {
	node := &intervalNode{start: start, end: end, cidr: cidr}
	if t.root == nil {
		t.root = node
		return nil
	}
	curr := t.root
	for {
		if end < curr.start {
			if curr.left == nil {
				curr.left = node
				return nil
			}
			curr = curr.left
		} else if start > curr.end {
			if curr.right == nil {
				curr.right = node
				return nil
			}
			curr = curr.right
		} else {
			return fmt.Errorf("overlapping intervals are not supported: [%d, %d] overlaps with [%d, %d]", start, end, curr.start, curr.end)
		}
	}
}

func (t *intervalTree) search(ip uint32) *CIDRRange {
	curr := t.root

	for curr != nil {
		if ip < curr.start {
			curr = curr.left
		} else if ip > curr.end {
			curr = curr.right
		} else {
			return curr.cidr
		}
	}

	return nil
}

// Build an interval tree from the CIDR ranges
type intervalNode struct {
	start, end  uint32
	left, right *intervalNode
	cidr        *CIDRRange
}

type intervalTree struct {
	root *intervalNode
}

func outputJSON(ips []string, filename string) error {
	type IP struct {
		Address string `json:"address"`
	}

	var data []IP
	for _, ip := range ips {
		data = append(data, IP{ip})
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer func() {
		cerr := file.Close()
		if err == nil {
			err = cerr
		}
	}()

	writer := bufio.NewWriter(file)
	_, err = writer.Write(jsonData)
	if err != nil {
		return err
	}

	return writer.Flush()
}

func outputCSV(ips []string, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		cerr := file.Close()
		if err == nil {
			err = cerr
		}
	}()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, ip := range ips {
		if err := writer.Write([]string{ip}); err != nil {
			return err
		}
	}

	return nil
}

func outputTerminal(ips []string) {
	for _, ip := range ips {
		fmt.Println(ip)
	}
}

func handleOutput(format string, ips []string, cidrListStr string) error {
	switch format {
	case "json":
		filename := fmt.Sprintf("ips_%s_%s.json", strings.ReplaceAll(cidrListStr, "/", "-"), time.Now().Format("2006-01-02T15-04-05"))
		return outputJSON(ips, filename)
	case "csv":
		filename := fmt.Sprintf("ips_%s_%s.csv", strings.ReplaceAll(cidrListStr, "/", "-"), time.Now().Format("2006-01-02T15-04-05"))
		return outputCSV(ips, filename)
	case "terminal":
		outputTerminal(ips)
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
