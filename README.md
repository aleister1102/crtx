# crtshx

`crtshx` is a powerful and concurrent subdomain enumeration tool that leverages Certificate Transparency logs from `crt.sh`. It is written in Go and designed for performance and flexibility, making it a valuable asset for security researchers and system administrators.

## Features

- **Concurrent Enumeration**: Utilizes goroutines to perform fast, parallel searches.
- **Multiple Input Methods**: Accepts domains via command-line flags, standard input (piping), or organization name.
- **Recursive Search**: A powerful deep search mode that first finds certificates for a domain, extracts all associated organization names, and then queries those organization names to uncover more related domains.
- **Flexible Output**: Prints a clean, unique, and sorted list of subdomains.
- **Easy to Use**: Simple and intuitive command-line interface with a comprehensive usage menu.

## Installation

### Install from GitHub
```sh
go install github.com/aleister1102/crtshx@latest
```

### Build from Source

To get started, ensure you have Go installed on your system. Then, you can build the tool from the source:

```sh
# Clone the repository (or just use the existing source code)
# git clone https://github.com/aleister1102/crtshx.git
# cd crtshx

# Build the executable
go build
```
This will create a `crtshx` (or `crtshx.exe` on Windows) executable in the current directory.

## Usage

`crtshx` provides a variety of flags and input methods to suit different enumeration needs.

### Basic Usage

**Find subdomains for a single domain:**
```sh
./crtshx -d example.com
```

**Find subdomains for multiple domains:**
```sh
./crtshx -d example.com -d anotherexample.com
```

**Read domains from `stdin` (e.g., from a file):**
```sh
cat list_of_domains.txt | ./crtshx
```

### Search by Organization

**Find all domains associated with an Organization Name:**
*Note: Use quotes if the name contains spaces.*
```sh
./crtx -o "Google LLC"
```

### Recursive Search

**Perform a deep, recursive search for a domain:**
This mode is significantly more thorough but also takes more time. It's great for discovering assets that might not be found through simple subdomain searches.
```sh
./crtshx -r -d example.com
```

### Adjusting Concurrency

**Use the `-c` flag to set the number of concurrent workers:**
The default is 50. Increasing this may speed up searches for large lists of domains, but be mindful of rate limits.
```sh
cat list_of_domains.txt | ./crtshx -c 100
```

**Using a custom blocklist file:**
Create a file (e.g., `my_blocklist.txt`) with one domain suffix per line. `crtx` will use this in addition to its default blocklist.
```sh
# my_blocklist.txt
# amazonaws.com
# digitalocean.com

./crtshx -d example.com -bf my_blocklist.txt
```

### Command-Line Options

```
crtshx - A powerful subdomain enumeration tool using crt.sh

Usage:
  crtshx [options]

Examples:
  cat domains.txt | crtshx
  crtshx -d example.com
  crtshx -d example.com -d anotherexample.com
  crtshx -o "Example Inc"
  crtshx -r -d example.com

Options:
  -bf string
        Path to a file containing additional domain suffixes to block
  -c int
        Set the concurrency level (default 50)
  -d value
        Domain to search for (can be specified multiple times)
  -o string
        Organization name to search for
  -r    Perform a recursive search (requires -d) 
