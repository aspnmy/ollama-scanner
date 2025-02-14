# Ollama Node Scanner Tool User Guide

## Tool Overview
- A tool for scanning Ollama nodes in a local area network, with the capability to automatically perform performance tests and export the results to a CSV file. It uses the zmap tool to scan IP addresses and checks each IP address for the Ollama service.

## Usage

### Basic Usage
- To run this tool from the command line, use the following basic command format:
  
```bash
  ./ollama-scanner [parameters]
```

### Parameter Description
| Parameter | Description | Default Value | 
| --- | --- | --- | 
| -gateway-mac | Specify the gateway MAC address, format: aa:bb:cc:dd:ee:ff | None (must be specified) | 
| -input | Input file path, the file content should be a list of IP addresses in CIDR format | ip.txt | 
| -output | CSV output file path | results.csv | 
| -no-bench | Disable performance benchmarking | false | 
| -prompt | Performance test prompt | Why does the sun shine? Answer in one sentence | 
| -T | zmap thread count | 10 |

### Usage Examples
- Scan with a specified gateway MAC address:

```bash
./ollama-scanner -gateway-mac aa:bb:cc:dd:ee:ff
```

- Scan with a specified gateway MAC address, disable performance tests, and specify output file:
  
```bash
./ollama-scanner -gateway-mac aa:bb:cc:dd:ee:ff -no-bench -output custom.csv
```

- Scan with a specified gateway MAC address and zmap thread count:

```bash
./ollama-scanner -gateway-mac aa:bb:cc:dd:ee:ff -T 20
```

### Tool Execution Process

#### Initialization: Parse command line parameters, create a cancellable context, check and install zmap (if not installed), initialize CSV writer, set up signal handling.
#### Scanning Process:
   Validate the input parameters.
- Execute zmap scan to get live IP addresses.
- Perform port check and Ollama service check on each live IP address.
- Retrieve model information on each IP address.
- If performance testing is not disabled, perform performance testing on each model.
#### Result Processing:
- Print scan results to the console.
- Write scan results to the CSV file.

### Precautions
- zmap Installation: The tool will attempt to install zmap automatically, but on some operating systems, manual installation may be required. If automatic installation fails, the tool will prompt you to install it manually and provide an installation link.
- Input File: The input file should contain a list of IP addresses in CIDR format. If the file does not exist, the tool will report an error.
- Performance Testing: Performance testing may consume significant time and resources. You can disable this feature using the -no-bench parameter.

## How to Compile
- Compile programs for all platforms: Run the make or make all command in the terminal to generate the corresponding executable files for macOS, Linux, and Windows platforms.
- Compile programs for a specific platform: ◦ Compile for macOS: make build-macos
- Compile for Linux: make build-linux
- Compile for Windows: make build-windows
- Clean generated files: Run the make clean command to delete all generated executable files.
- Ensure that your system has the Go environment installed and that the go command can be used in the terminal.

## Communication and Discussion
- https://t.me/+YfCVhGWyKxoyMDhl
