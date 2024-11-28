# GCP Enumerator

is a Go-based tool designed to scan Google Cloud Storage (GCS) buckets for existence and accessibility. It leverages keyword-based permutations or custom wordlists to identify potential bucket names, checking their status and attempting to list accessible objects.

Features
--------

- Keyword-Based Permutations: Generate bucket names based on a single keyword or multiple keywords from a file.
- Custom Wordlists: Use your own suffix wordlist or the default wordlist for permutations.
- Concurrency Control: Specify the number of concurrent requests to balance speed and resource usage.
- Output Logging: Save results to a file for later review.
- Verbose Mode: Get detailed feedback on request responses and errors.
- Bucket Content Enumeration: List objects in buckets with accessible permissions.

Installation
------------

`go install github.com/Vulnpire/gcpenum@latest`

Usage
-----

Basic Command:

`gcpenum -n <keyword>`

Options:
- `-n`: Single keyword for bucket name permutations (e.g., `-n example`).
- `-l`: File path containing multiple keywords (one per line) (e.g., `-l keywords.txt`).
- `-w`: Custom wordlist file for suffixes, defaults to a downloaded wordlist (e.g., `-w custom-wordlist.txt`).
- `-o`: Save output results to a file (e.g., `-o results.txt`).
- `-c`: Number of concurrent requests (default: 10) (e.g., `-c 20`).
- `-v`: Enable verbose mode for detailed logs (e.g., `-v`).

Examples
--------

1. Single Keyword:
   `gcpenum -n mycompany`

2. Multiple Keywords:
   `gcpenum -l keywords.txt -c 15 -o results.txt`

3. Custom Wordlist:
   `gcpenum -n project -w my-wordlist.txt`

Wordlist Management
-------------------

The tool downloads a default wordlist to:
`~/.config/gcpenum/words.txt`

If missing, the file will be downloaded again during execution. A custom wordlist can be provided with the `-w` flag.


## Acknowledgments

    Inspired by the need for secure cloud storage enumeration.
    Default wordlist curated with common naming patterns and creative suffixes.

Enjoy enumerating buckets responsibly! 🚀
