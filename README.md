# cosmos-dump

`cosmos-dump` is a command-line tool that connects to a gRPC server and extracts blockchain data to JSON files.

Tested with Cosmos SDK v0.50.x.

## Features

- Extract block and transaction chain data to JSON files.
- Connects to a gRPC server.
- Supports custom start and stop block heights.
- Outputs data to specified directories.
- Supports server reflection; no need to specify the proto file.
- `Any` type are properly decoded.

## Installation

To install the `cosmos-dump` tool, you need to have Go installed on your system. Then, you can use the following command to install the tool:

```sh
go install github.com/liftedinit/cosmos-dump/cmd/cosmos-dump@latest
```

## Usage
The basic usage of the cosmos-dump tool is as follows:
```shell
cosmos-dump [address] [flags]
```

## Flags

- `-s`, `--start` - The starting block height to extract data from (default: 1)
- `-e`, `--stop` - The stopping block height to extract data from (default: 1)
- `-o`, `--out` - The output directory to store the extracted data (default: "out")

## Example

```shell
cosmos-dump localhost:9090 -s 1 -e 100 -o ./data
```

This command will connect to the gRPC server running on `localhost:9090`, extract data from block height `1` to `100`, and store the extracted data in the `data` directory.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Disclaimer

This software is provided "as is", without warranty of any kind, express or implied, including but not limited to the warranties of merchantability, fitness for a particular purpose, and noninfringement. In no event shall the authors or copyright holders be liable for any claim, damages, or other liability, whether in an action of contract, tort, or otherwise, arising from, out of, or in connection with the software or the use or other dealings in the software.
