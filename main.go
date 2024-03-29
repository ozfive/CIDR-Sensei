package main

import (
	"bufio"
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
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
	length int
}

func parseCIDRList(cidrList []string) ([]CIDRRange, error) {
	cidrRanges := make([]CIDRRange, len(cidrList))
	for i, cidrStr := range cidrList {
		_, ipNet, err := net.ParseCIDR(cidrStr)
		if err != nil {
			return nil, fmt.Errorf("error parsing CIDR %s: %s", cidrStr, err)
		}
		start := ip2uint(ipNet.IP)
		end := start | ^ip2uint(net.IP(ipNet.Mask))
		cidrRanges[i] = CIDRRange{ipNet: ipNet, start: start, end: end, length: int(end - start)}
	}
	return cidrRanges, nil
}

func ip2uint(ip net.IP) uint32 {
	return binary.BigEndian.Uint32(ip.To4())
}

func uint2ip(ip uint32) net.IP {
	result := make(net.IP, 4)
	binary.BigEndian.PutUint32(result, ip)
	return result
}

func cidrToIPsParallelBinarySearch(cidrRanges []CIDRRange, concurrency int) ([]string, error) {
	var ips []string
	semaphoreChan := make(chan struct{}, concurrency)
	ipChan := make(chan []byte, concurrency)
	errChan := make(chan error, 1) // Channel for sending errors

	// Pre-allocate a pool of byte slices for storing IP addresses
	ipPool := sync.Pool{
		New: func() interface{} {
			return make([]byte, 15)
		},
	}

	// Sort the CIDR ranges by their starting IPs
	sort.Slice(cidrRanges, func(i, j int) bool {
		return cidrRanges[i].start < cidrRanges[j].start
	})

	// Process each CIDR range in a separate goroutine
	var wg sync.WaitGroup
	for i, cidrRange := range cidrRanges {
		wg.Add(1)
		go func(cidr CIDRRange, idx int) {
			defer wg.Done()
			semaphoreChan <- struct{}{}
			start := cidr.start
			for i := uint32(0); i < uint32(cidr.length); i++ {
				ip := uint2ip(start + i)
				// Use binary search to find the index of the last CIDR range that contains the current IP
				lastIndex := sort.Search(len(cidrRanges), func(j int) bool {
					return cidrRanges[j].end >= ip2uint(ip)
				}) - 1
				for j := idx; j <= lastIndex; j++ {
					if ip2uint(ip) <= cidrRanges[j].end {
						if value := cidrRanges[j].ipNet.String(); value != "" {
							ipBytes := ipPool.Get().([]byte)
							copy(ipBytes, ip.String())
							ipChan <- ipBytes
						}
						break
					}
				}
			}
			<-semaphoreChan
		}(cidrRange, i)
	}

	// Wait for all goroutines to finish and close the IP channel
	go func() {
		wg.Wait()
		close(ipChan)
	}()

	// Receive IP addresses from the channel and add them to the result slice
	for ipBytes := range ipChan {
		ipString := string(ipBytes)
		ips = append(ips, ipString)
		// Put the byte slice back in the pool
		ipPool.Put(ipBytes)
	}

	// Check for any errors
	select {
	case err := <-errChan:
		return nil, err
	default:
		return ips, nil
	}
}

type intervalNode struct {
	start, end  uint32
	left, right *intervalNode
	cidr        *CIDRRange
}

type intervalTree struct {
	root *intervalNode
}

func (t *intervalTree) insert(start, end uint32, cidr *CIDRRange) {
	node := &intervalNode{start: start, end: end, cidr: cidr}
	if t.root == nil {
		t.root = node
		return
	}
	curr := t.root
	for {
		if end <= curr.start {
			if curr.left == nil {
				curr.left = node
				return
			}
			curr = curr.left
		} else if start >= curr.end {
			if curr.right == nil {
				curr.right = node
				return
			}
			curr = curr.right
		} else {
			panic("overlapping intervals are not supported")
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

func cidrToIPsParallelIntervalTree(cidrRanges []CIDRRange, concurrency int) ([]string, error) {
	var ips []string
	semaphoreChan := make(chan struct{}, concurrency)
	ipChan := make(chan []byte, concurrency)
	var wg sync.WaitGroup
	var err error

	// Pre-allocate a pool of byte slices for storing IP addresses
	ipPool := sync.Pool{
		New: func() interface{} {
			return make([]byte, 15)
		},
	}

	// Build an interval tree from the CIDR ranges
	tree := intervalTree{}
	for _, cidrRange := range cidrRanges {
		tree.insert(cidrRange.start, cidrRange.end, &cidrRange)
	}

	for _, cidrRange := range cidrRanges {
		wg.Add(1)

		go func(cidr CIDRRange) {
			defer wg.Done()

			semaphoreChan <- struct{}{}

			start := cidr.start

			for i := uint32(0); i < uint32(cidr.length); i++ {
				ip := start + i

				if c := tree.search(ip); c != nil {
					if value := c.ipNet.String(); value != "" {
						ipBytes := ipPool.Get().([]byte)

						copy(ipBytes, uint2ip(ip).String())

						ipChan <- ipBytes
					}
				}
			}
			<-semaphoreChan
		}(cidrRange)
	}

	go func() {
		wg.Wait()
		close(ipChan)
	}()

	// Receive IP addresses from the channel and add them to the result slice
	for ipBytes := range ipChan {
		ipString := string(ipBytes)
		ips = append(ips, ipString)

		// Put the byte slice back in the pool
		ipPool.Put(ipBytes)
	}

	if err != nil {
		return nil, err
	}

	return ips, nil
}

/*
func binarySearch(cidrRanges []CIDRRange, ip uint32) string {
	left := 0
	right := len(cidrRanges) - 1
	for left <= right {
		mid := (left + right) / 2
		if ip < cidrRanges[mid].start {
			right = mid - 1
		} else if ip > cidrRanges[mid].end {
			left = mid + 1
		} else {
			return cidrRanges[mid].ipNet.String()
		}
	}
	return ""
}
*/

func cidrToIPs(cidrRanges []CIDRRange, parallel bool, concurrency int, algorithm string) ([]string, error) {
	if parallel {
		if algorithm == "interval-tree" {
			return cidrToIPsParallelIntervalTree(cidrRanges, concurrency)
		} else if algorithm == "binary-search" {
			return cidrToIPsParallelBinarySearch(cidrRanges, concurrency)
		} else {
			// Default to binary search
			return cidrToIPsParallelBinarySearch(cidrRanges, concurrency)
		}
	} else {
		return cidrToIPsBinarySearch(cidrRanges)
	}

}

func cidrToIPsBinarySearch(cidrRanges []CIDRRange) ([]string, error) {
	// Prepare the sorted list of CIDR ranges
	sortedCIDRRanges := make([]CIDRRange, len(cidrRanges))

	copy(sortedCIDRRanges, cidrRanges)

	sort.Slice(sortedCIDRRanges, func(i, j int) bool {
		return sortedCIDRRanges[i].start < sortedCIDRRanges[j].start
	})

	// Expand the CIDR ranges into a list of IPs using binary search to find the region for each IP
	var ips []string

	for _, cidrRange := range sortedCIDRRanges {
		for i := uint32(0); i < uint32(cidrRange.length); i++ {
			ip := uint2ip(cidrRange.start + i)
			idx := sort.Search(len(sortedCIDRRanges), func(j int) bool {
				return sortedCIDRRanges[j].end >= ip2uint(ip)
			})
			if idx < len(sortedCIDRRanges) && sortedCIDRRanges[idx].start <= ip2uint(ip) {
				ips = append(ips, ip.String())
			}
		}
	}

	return ips, nil
}

func main() {
	var outputFormat string
	var cidrListStr string
	var parallel bool
	var concurrency int
	var algorithm string

	flag.StringVar(&outputFormat, "output", "terminal", "the output format (json, csv, or terminal)")
	flag.StringVar(&cidrListStr, "cidr", "", "a comma-separated list of CIDR blocks to expand into IPs")
	flag.BoolVar(&parallel, "parallel", false, "enable parallel processing")
	flag.IntVar(&concurrency, "concurrency", defaultConcurrency, "set the number of workers for parallel processing")
	flag.StringVar(&algorithm, "algorithm", defaultAlgorithm, "the algorithm to use for expanding CIDR blocks into IPs (binary-search, interval-tree)")
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

	if cidrListStr == "" {
		fmt.Println("Error: -cidr is required.")
		flag.Usage()
		os.Exit(1)
	}

	cidrList := strings.Split(cidrListStr, ",")
	startTime := time.Now()
	cidrRanges, err := parseCIDRList(cidrList)

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		flag.Usage()
		os.Exit(1)
	}

	ips, err := cidrToIPs(cidrRanges, parallel, concurrency, algorithm)

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		flag.Usage()
		os.Exit(1)
	}

	switch outputFormat {
	case "json":
		filename := fmt.Sprintf("ips_%s_%s.csv", strings.ReplaceAll(cidrListStr, "/", "-"), time.Now().Format("2006-01-02T15-04-05"))
		err = outputJSON(ips, filename)
		if err != nil {
			fmt.Printf("Error writing JSON output to file: %v\n", err)
		}
	case "csv":
		filename := fmt.Sprintf("ips_%s_%s.csv", strings.ReplaceAll(cidrListStr, "/", "-"), time.Now().Format("2006-01-02T15-04-05"))
		err := outputCSV(ips, filename)
		if err != nil {
			fmt.Printf("Error writing CSV file: %v\n", err)
		}
	default:
		outputTerminal(ips)
	}

	fmt.Printf("Took %v seconds to complete.\n", time.Since(startTime).Seconds())
}

func outputJSON(ips []string, filename string) error {
	type IP struct {
		Address string `json:"address"`
	}

	var data []IP
	for _, ip := range ips {
		data = append(data, IP{ip})
	}

	jsonData, err := json.MarshalIndent(data, "", " ")
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

	err = writer.Flush()
	if err != nil {
		return err
	}

	return nil
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

	writer.Flush()

	return nil
}

func outputTerminal(ips []string) {
	for _, ip := range ips {
		fmt.Println(ip)
	}
}
