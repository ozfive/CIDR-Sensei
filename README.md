# CIDR-Sensei

CIDR-Sensei is a command-line tool that expands a comma-separated list of CIDR blocks into a list of IP addresses. It supports both sequential and parallel processing.
# Implementation

CIDR-Sensei is a tool written in Go that helps you easily expand a list of CIDR blocks into a list of IP addresses. With the -concurrency flag, you can run the program in parallel to speed up the expansion process.

To use it, simply provide a comma-separated list of CIDR blocks to the -cidr flag, and CIDR-Sensei will do the rest. It first parses the list and stores the start and end IP addresses of each CIDR block in a slice of CIDRRange structs.

Next, it expands the CIDR blocks into a list of IP addresses. It can do this in two ways: either by using a binary search to find the CIDR block that contains each IP address, or by splitting the CIDR blocks into smaller chunks and processing each chunk in parallel. The resulting list of IP addresses is output to the console.

So whether you're a network administrator or just curious about IP addresses, CIDR-Sensei has got you covered!

# Installation

Clone the repository and build the tool using the following commands:

```console
git clone https://github.com/your_username/cidr-sensei.git
cd cidr-sensei
go build -o cidr-sensei cmd/cidr-sensei/main.go
```

# Usage

```shell
./cidr-sensei -cidr="10.0.0.0/8,172.16.0.0/12,192.168.0.0/16" -parallel -concurrency=100

```
You can use the following options:

*    **-cidr**: a comma-separated list of CIDR blocks to expand into IP addresses (required).
*    **-parallel**: enables parallel processing (optional).
*    **-concurrency**: sets the number of workers for parallel processing (default is 100, optional).

# Example
```console
./cidr-sensei -cidr="10.0.0.0/8,172.16.0.0/12,192.168.0.0/16" -parallel -concurrency=100
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

The above command will expand the CIDR blocks **10.0.0.0/8**, **172.16.0.0/12**, and **192.168.0.0/16** into a list of IP addresses, using 100 workers for parallel processing.

# Dependencies

*   Go v1.16 or later
