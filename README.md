# CIDR-Sensei
[![DeepSource](https://app.deepsource.com/gh/ozfive/CIDR-Sensei.svg/?label=active+issues&show_trend=true&token=_FFNSjcgffdEw4DWcLU42oRJ)](https://app.deepsource.com/gh/ozfive/CIDR-Sensei/) [![DeepSource](https://app.deepsource.com/gh/ozfive/CIDR-Sensei.svg/?label=resolved+issues&show_trend=true&token=_FFNSjcgffdEw4DWcLU42oRJ)](https://app.deepsource.com/gh/ozfive/CIDR-Sensei/)

[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=ozfive_CIDR-Sensei&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=ozfive_CIDR-Sensei)
[![Bugs](https://sonarcloud.io/api/project_badges/measure?project=ozfive_CIDR-Sensei&metric=bugs)](https://sonarcloud.io/summary/new_code?id=ozfive_CIDR-Sensei)
[![Code Smells](https://sonarcloud.io/api/project_badges/measure?project=ozfive_CIDR-Sensei&metric=code_smells)](https://sonarcloud.io/summary/new_code?id=ozfive_CIDR-Sensei)
[![Duplicated Lines (%)](https://sonarcloud.io/api/project_badges/measure?project=ozfive_CIDR-Sensei&metric=duplicated_lines_density)](https://sonarcloud.io/summary/new_code?id=ozfive_CIDR-Sensei)
[![Lines of Code](https://sonarcloud.io/api/project_badges/measure?project=ozfive_CIDR-Sensei&metric=ncloc)](https://sonarcloud.io/summary/new_code?id=ozfive_CIDR-Sensei)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=ozfive_CIDR-Sensei&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=ozfive_CIDR-Sensei)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=ozfive_CIDR-Sensei&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=ozfive_CIDR-Sensei)
[![Technical Debt](https://sonarcloud.io/api/project_badges/measure?project=ozfive_CIDR-Sensei&metric=sqale_index)](https://sonarcloud.io/summary/new_code?id=ozfive_CIDR-Sensei)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=ozfive_CIDR-Sensei&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=ozfive_CIDR-Sensei)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=ozfive_CIDR-Sensei&metric=vulnerabilities)](https://sonarcloud.io/summary/new_code?id=ozfive_CIDR-Sensei)

CIDR-Sensei is a command-line tool that expands a comma-separated list of CIDR blocks into a list of IP addresses. It supports both sequential and parallel processing.

## **Implementation**

CIDR-Sensei is a tool written in Go that helps you easily expand a list of CIDR blocks into a list of IP addresses. With the `-concurrency` flag, you can run the program in parallel to speed up the expansion process while minimizing memory usage.

To use it, simply provide a comma-separated list of CIDR blocks to the `-cidr` flag, and CIDR-Sensei will do the rest. It first parses the list and stores the start and end IP addresses of each CIDR block in a slice of `CIDRRange` structs.

Next, it expands the CIDR blocks into a list of IP addresses. It can do this in two ways: either by using a binary search to find the CIDR block that contains each IP address or by utilizing an interval tree for efficient range queries. This release includes both binary search and interval tree algorithms, selectable via the `-algorithm` flag. The resulting list of IP addresses are streamed directly to the terminal, CSV, or JSON with the `-output` option.

### **Benefits:**

- **Lower Memory Usage:** Streams IP addresses directly to the output, avoiding the need to store them all in memory.
- **Enhanced Performance:** Potentially faster processing as it eliminates the overhead of appending to a large slice.
- **Flexible Output:** Supports JSON, CSV, and terminal outputs, catering to various use cases.

So whether you're a network administrator or just curious about IP addresses, CIDR-Sensei has got you covered!

## **Installation**

Clone the repository and build the tool using the following commands:

```console
git clone https://github.com/your_username/cidr-sensei.git
cd cidr-sensei
go build -o cidr-sensei cmd/cidr-sensei/main.go
```

Binary releases are available [HERE](https://github.com/ozfive/CIDR-Sensei/tags) for many platforms.

# Usage

```shell
./cidr-sensei -output="json" -cidr="10.0.0.0/8,172.16.0.0/12,192.168.0.0/16" -parallel -concurrency=100 -algorithm="interval-tree"

```
You can use the following options:
*    **-output**: Sets the output format ("json", "csv", or "terminal") (required).
*    **-cidr**: A comma-separated list of CIDR blocks to expand into IP addresses (required).
*    **-parallel**: Enables parallel processing (optional).
*    **-concurrency**: Sets the number of workers for parallel processing (default=100, optional).
*    **-algorithm**: Sets the algorithm to use when parallel processing. ("binary-search", "interval-tree") (default="binary-search" optional)

# Example
```console
./cidr-sensei -output="json" -cidr="10.0.0.0/8,172.16.0.0/12,192.168.0.0/16" -parallel -concurrency=100 -algorithm="interval-tree"
10.0.0.0
10.0.0.1
10.0.0.2
10.0.0.3
...
192.168.255.252
192.168.255.253
192.168.255.254
192.168.255.255
Took 0.372257 seconds to complete.
```

The above command will expand the CIDR blocks **10.0.0.0/8**, **172.16.0.0/12**, and **192.168.0.0/16** into a list of IP addresses in a JSON file, using 100 workers for parallel processing and the interval-tree algorithm when -parallel is used.

# Dependencies

*   Go v1.16 or later
