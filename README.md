# Helios-Core [![codecov](https://codecov.io/gh/Helios-Chain-Labs/helios-core/branch/dev/graph/badge.svg?token=WTDFT58GB8)](https://codecov.io/gh/Helios-Chain-Labs/helios-core)

![Banner!](assets/logo.png)

[//]: # ([![Project Status: Active -- The project has reached a stable, usable)
[//]: # (state and is being actively)
[//]: # (developed.]&#40;https://img.shields.io/badge/repo%20status-Active-green.svg?style=flat-square&#41;]&#40;https://www.repostatus.org/#active&#41;)
[//]: # ([![GoDoc]&#40;https://img.shields.io/badge/godoc-reference-blue?style=flat-square&logo=go&#41;]&#40;https://pkg.go.dev/github.com/Helios-Chain-Labs/sdk-go/chain&#41;)
[//]: # ([![Discord]&#40;https://badgen.net/badge/icon/discord?icon=discord&label&#41;]&#40;https://discord.gg/helios&#41;)


Home of the following services:

* [heliades](/cmd/heliades)

## Architecture

<img alt="architecture.png" src="./assets/architecture.png" width="100%"/>

## Installation

### Building from sources

In order to build from source you’ll need at least [Go 1.22+](https://golang.org/dl/).

```bash
# need to clone if you plan to run tests, and use Makefile
$ git clone git@github.com:Helios-Chain-Labs/helios-core.git
$ cd helios-core
$ make install

# or simply do this to fetch modules and build executables
$ go install github.com/Helios-Chain-Labs/helios-core/cmd/...
```
### Quick Setup
The most convenient way to launch services is by running the setup script:
```bash
$ ./setup.sh
```
Then run an instance of the heliades node.
```bash
$ ./heliades.sh
```

Voila! You have now successfully setup a full node on the Helios Chain.

## Generating the module specification docs
```bash
$ cd docs && yarn && yarn run serve
```
## Generating REST and gRPC Gateway docs
First, ensure that the `Enable` and `Swagger` values are true in APIConfig set in `cmd/heliades/config/config.go`.

Then simply run the following command to auto-generate the Swagger UI docs.
```bash
$ make proto-swagger-gen
```
Then when you start the Helios Daemon, simply navigate to [http://localhost:10337/swagger/](http://localhost:10337/swagger/).

## Generating Helios Chain API gRPC Typescript bindings

```bash
$ make gen
```
Then when you start the Helios Daemon, simply navigate to [http://localhost:10337/swagger/](http://localhost:10337/swagger/).


## Maintenance

To run all unit tests:

```bash
$ go test ./helios-chain/...
```
