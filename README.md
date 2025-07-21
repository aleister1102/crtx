# crtshx

A fast, concurrent subdomain enumeration tool using Certificate Transparency logs from crt.sh.

## Features

- **Concurrent searches** with configurable worker threads
- **Multiple input methods**: CLI flags, stdin, or organization name
- **Recursive mode** for deep subdomain discovery
- **Verbose logging** for debugging and monitoring

## Installation

```bash
go install github.com/aleister1102/crtshx@latest
```

Or build from source:
```bash
go build
```

## Usage

### Basic Examples

```bash
# Single domain
./crtshx -d example.com

# Multiple domains
./crtshx -d example.com -d another.com

# From file
cat domains.txt | ./crtshx

# Organization search
./crtshx -o "Google LLC"

# Recursive search (thorough but slower)
./crtshx -r -d example.com

# Verbose output
./crtshx -v -d example.com

# Custom concurrency
./crtshx -c 100 -d example.com
```

## Options

```
-d value    Domain to search (can be used multiple times)
-o string   Organization name to search  
-r          Recursive search (requires -d)
-c int      Concurrency level (default: 50)
-v          Verbose output
-bf string  Custom blocklist file
```

## License

MIT 
