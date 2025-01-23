# `Hyperion`

## Abstract

The hyperion module enables the Helios Chain to support a trustless, on-chain bidirectional ERC-20 token bridge to Ethereum. In this system,
holders of ERC-20 tokens on Ethereum can convert their ERC-20 tokens to Cosmos-native coins on
the Helios Chain and vice-versa.  

This decentralized bridge is secured and operated by the validators of the Helios Chain.

## Contents

1. **[Definitions](./01_definitions.md)**
2. **[Workflow](./02_workflow.md)**    
3. **[State](./03_state.md)** 
4. **[Messages](./04_messages.md)**
5. **[Slashing](./05_slashing.md)**
6. **[End-Block](./06_end_block.md)**
7. **[Events](./07_events.md)**
8. **[Parameters](./08_params.md)**

### Components

1. **[Hyperion](https://etherscan.io/address/0xF955C57f9EA9Dc8781965FEaE0b6A2acE2BAD6f3) smart contract on Ethereum**
2. **Hyperion module on the Helios Chain**
3. **[Peggo](https://github.com/Helios-Chain-Labs/peggo) (off-chain relayer aka orchestrator)**
    - **Oracle** (Observes events of Hyperion contract and send claims to the Hyperion module)
    - **EthSigner** (Signs Valset and Batch confirmations to the Hyperion module)
    - **Batch Requester** (Sends batch token withdrawal requests to the Hyperion module)
    - **Valset Relayer** (Submits Validator set updates to the Hyperion contract)
    - **Batch Relayer** (Submits batches of token withdrawals to the Hyperion contract)

In addition to running an `heliades` node to sign blocks, Helios Chain validators must also run the `peggo` orchestrator to relay data from the Hyperion smart contract on Ethereum and the Hyperion module on the Helios Chain.

### Peggo Functionalities

1. **Maintaining an up-to-date checkpoint of the Helios Chain validator set on Ethereum**
2. **Transferring ERC-20 tokens from Ethereum to the Helios Chain**
3. **Transferring pegged tokens from the Helios Chain to Ethereum**
