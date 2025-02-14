
# Ollama Scanner Node Scanning Tool Usage Guide

![Ollama Scanner](../images/README/1739551751297.png)

[English](README_en.md) · [简体中文](../README.md)

## Tool Overview

- A tool for scanning Ollama Scanner nodes in a local area network, with the ability to automatically perform performance tests and export the results to a CSV file. It utilizes the zmap tool to scan IP addresses and performs various checks on each IP address.
- Can also be used to sniff public IPs for the presence of Ollama services.

## Usage

### Using on Windows

Since zmap cannot be installed on Windows, there are several ways to use this tool on Windows:

- Use the masscan version of the sniffer (more versatile - recommended)
- Use the WSL mode to run the Linux version of the sniffer
- Use a Docker container

  ```docker
  # Download the image
  docker pull docker.io/aspnmy/ollama_scanner:v2.2-zmap
  # Run the sniffer
  docker exec -it [dockerid] /usr/local/bin/ollama_scanner [parameters]
  ```

### Basic Usage

- To run this tool from the command line, you can use the following basic command format:

```bash
./ollama_scanner [parameters]
```

### Parameter Description

| Parameter    | Description                                      | Default Value                    |
| ------------ | ------------------------------------------------ | -------------------------------- |
| -gateway-mac | Specify the gateway MAC address in the format aa:bb:cc:dd:ee:ff | None (must be specified)         |
| -input       | Path to the input file, containing a list of IP addresses in CIDR format | ip.txt                           |
| -output      | Path to the CSV output file                      | results.csv                      |
| -no-bench    | Disable performance benchmarking                 | false                            |
| -prompt      | Performance test prompt                          | Why does the sun shine? Answer in one sentence |
| -T           | Number of zmap threads                           | 10                               |

### Usage Examples

- Specify IP address, disable performance test, specify output file, and set zmap thread count:

```bash
./ollama_scanner -input ip.txt -no-bench -output custom.csv -T 20
```

- Scan a specified list of IP addresses:

```bash
./ollama_scanner -input ip.txt
```

- Scan with a specified gateway MAC address:

```bash
./ollama_scanner -gateway-mac aa:bb:cc:dd:ee:ff
```

- Specify gateway MAC address, disable performance test, and specify output file:

```bash
./ollama_scanner -gateway-mac aa:bb:cc:dd:ee:ff -no-bench -output custom.csv
```

- Specify gateway MAC address and zmap thread count:

```bash
./ollama_scanner -gateway-mac aa:bb:cc:dd:ee:ff -T 20
```

### Tool Execution Process

#### Initialization: Parse command line parameters, create a cancelable context, check and install zmap (if not installed), initialize CSV writer, and set signal handling.

#### Scanning Process:

- Validate the input parameters.
- Perform zmap scan to get live IP addresses.
- Perform port checks and Ollama service checks on each live IP address.
- Retrieve model information on each IP address.
- If performance benchmarking is not disabled, perform performance tests on each model.

#### Result Processing:

- Print scan results to the console.
- Write scan results to a CSV file.

### Notes

- zmap or masscan installation: The tool will attempt to automatically install zmap or masscan, but on some operating systems, manual installation may be required. If automatic installation fails, the tool will prompt you to install it manually and provide instructions.
- Input file: The input file must contain a list of IP addresses in CIDR format. If the file does not exist, the tool will report an error.
- Performance testing: Performance testing may consume significant time and resources. You can disable this feature using the -no-bench parameter.

## How to Compile the Program

- Add compilation for arm64 platform sniffer. The arm64 architecture program can be run directly or a Docker image can be pulled.
- Compile the program for all platforms: Run the `make` or `make all` command in the terminal to generate executable files for macOS, Linux, and Windows platforms.
- Compile the program for a specific platform:

  - Compile for macOS: `make build-macos`
  - Compile for Linux: `make build-linux`
  - Compile for Windows: `make build-windows`
- Clean the generated files: Run the `make clean` command to delete all generated executable files.
- Ensure that your system has the Go environment installed and the `go` command can be used in the terminal.

## How to Compile the Docker Image

- Clone this project and modify the `env.json.def` file to `env.json`, modifying the parameters inside.

  ```bash
  mv env.json.def env.json
  ```
- Run the bash script `build.sh`

  ```bash
  bash build.sh
  ```

## Communication

- https://t.me/+YfCVhGWyKxoyMDhl
```
