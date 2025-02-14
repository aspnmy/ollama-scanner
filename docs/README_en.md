# Ollama Scanner Node Scanning Tool User Guide

![Ollama Scanner](../images/README/1739551751297.png)

[English](README_en.md) · [简体中文](README.md)

## Branch Description

- `webUI_mongodb`: Mainly stores exported data in MongoDB database in JSON format
- Supports reading MongoDB database files via WebUI
- WebUI is developed using Vue, supporting real-time rendering of results
- Node.js version manager recommended to use fnm for management

## Tool Overview

- A tool for scanning Ollama Scanner nodes in a local area network, capable of automatically performing performance tests and exporting results to CSV files. It uses the zmap tool to scan IP addresses, and checks each IP for Ollama services.
- Can also be used to sniff public IPs for Ollama services

## Usage

### How to Get Gateway MAC Address

- How to get gateway MAC address:

```bash
ip addr show | grep link/ether
```

- Or enter `ip addr show` and check the MAC address of the network card that accesses the external network

### Support WebUI Access Mode

- Export scan results to liteSQL database
- Support querying scan results in WebUI form

### Usage on Windows

Since zmap cannot be installed on Windows, the following usage methods are available:

- Use masscan version of the sniffer (more general - recommended)
- Use WSL mode to run the Linux version of the sniffer
- Use Docker container

  ```docker
  # Download image
  docker pull docker.io/aspnmy/ollama_scanner:v2.2-zmap
  # Run sniffer
  docker exec -it [dockerid] /usr/local/bin/ollama_scanner [parameters]
  ```

### Basic Usage

- When running the tool in the command line, use the following basic command format:

```bash
./ollama_scanner [parameters]
```

### Parameter Description

| Parameter    | Description                                      | Default Value                   |
| ------------ | ------------------------------------------------ | ------------------------------ |
| -gateway-mac | Specify the gateway MAC address, format: aa:bb:cc:dd:ee:ff | None (must specify)              |
| -input       | Input file path, file content is a list of IP addresses in CIDR format | ip.txt                         |
| -output      | CSV output file path                             | results.csv                    |
| -no-bench    | Disable performance benchmark test               | false                          |
| -prompt      | Performance test prompt                          | Why does the sun shine? Answer in one sentence |
| -T           | Number of zmap threads                           | 10                             |

### Usage Examples

- Specify IP address, disable performance test, specify output file, and specify zmap threads:

```bash
./ollama_scanner -gateway-mac aa:bb:cc:dd:ee:ff -input ip.txt -no-bench -output custom.csv -T 20
```

- Scan with specified IP address list:

```bash
./ollama_scanner -gateway-mac aa:bb:cc:dd:ee:ff -input ip.txt
```

- Scan with specified gateway MAC address:

```bash
./ollama_scanner -gateway-mac aa:bb:cc:dd:ee:ff
```

- Scan with specified gateway MAC address, disable performance test, and specify output file:

```bash
./ollama_scanner -gateway-mac aa:bb:cc:dd:ee:ff -no-bench -output custom.csv
```

- Scan with specified gateway MAC address and zmap threads:

```bash
./ollama_scanner -gateway-mac aa:bb:cc:dd:ee:ff -T 20
```

### Tool Execution Process

#### Initialization: Parse command line parameters, create cancellable context, check and install zmap (if not installed), initialize CSV writer, set signal processing.

#### Scanning Process:

- Validate input parameters.
- Execute zmap scan to get live IP addresses.
- Check ports and Ollama services on each live IP address.
- Get model information on each IP address.
- If performance test is not disabled, perform performance test on each model.

#### Result Processing:

- Print scan results to the console.
- Write scan results to CSV file.

### Notes

- zmap or masscan installation: The tool will attempt to automatically install zmap or masscan, but manual installation may be required on some operating systems. If automatic installation fails, the tool will prompt you to manually install and provide installation instructions.
- Input file: The input file must contain a list of IP addresses in CIDR format. If the file does not exist, the tool will report an error.
- Performance test: Performance tests may consume a lot of time and resources. You can disable this feature using the `-no-bench` parameter.

## How to Compile the Program

- v2.2.3 adds MongoDB driver. If MongoDB is not located on the local machine during compilation, you can specify the access entry in `env.json`. The default access value is "localhost:27017".
- v2.2.3_docker defaults to MongoDB version v4.4.0 during deployment, with localhost:27017 as the default access entry.
- Added support for compiling arm64 platform sniffer. The arm64 architecture program can be run directly or docker image can be pulled directly.
- Compile the program for all platforms: Run `make` or `make all` commands in the terminal to generate executable files for macOS, Linux, and Windows platforms respectively.
- Compile the program for a specific platform:

  - Compile for macOS: `make build-macos`
  - Compile for Linux: `make build-linux`
  - Compile for Windows: `make build-windows`
- Clean the generated files: Run `make clean` command to delete all generated executable files.
- Ensure that your system has Go environment installed and the `go` command can be used in the terminal.

## How to Compile Docker Image

- Clone the project, rename `env.json.def` file to `env.json`, and modify the parameters inside

  ```bash
  mv env.json.def env.json
  ```
- Run the `build.sh` script

  ```bash
  bash build.sh
  ```

## Communication and Discussion

- https://t.me/Ollama_Scanner
