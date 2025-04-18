---
sidebar_position: 1
title: Definitions
---


# Definitions

Words matter and we seek clarity in the terminology, so we can have clarity in our thinking and communication.
Key concepts that we mention below are defined here:

- `Operator` - This is a person (or people) who control an Helios Chain validator node. This is also called `valoper` or "Validator Operator" in the Cosmos SDK staking module. 
- `Validator` - This is an Helios Chain validating node (signs blocks)
- `Orchestrator` - This is the off-chain `hyperion` service which performs the following roles for the `Operator`:
  - `Eth Signer` -  Signs transactions used to move tokens between the two chains using Ethereum private keys. 
  - `Oracle` - Signs `Claims` using Helios Chain account private keys which are submitted to the Hyperion module where they are then aggregated into `Attestations`.
  - `Relayer` - Submits Valset updates and Batch transactions to the Hyperion contract on Ethereum. It earns fees from the transactions in a batch.
- `Validator Set` - The set of Helios Chain validators, along with their respective voting power as determined by their stake weight, also referred to as a Valset. These are ed25519 public keys (prefixed by`heliosvalcons`) used to sign Tendermint blocks.
- `Claim` - an Ethereum event signed and submitted to Helios by a single `Orchestrator`
- `Attestation` - an aggregation of claims that eventually becomes `observed` by all orchestrators.
- `Hyperion Contract` - The Ethereum contract that holds all of the ERC-20 tokens. It also maintains a compressed checkpointed representation of the Helios Chain validator set using `Delegate Keys` and normalized powers. For example if a validator has 5% of the Helios Chain validator power, their delegate key will have 5% of the voting power in the `Hyperion Contract`. These values are regularly updated on the contract to keep the Valset checkpoint in sync with the real Helios Chain validator set. 
- `Hyperion Tx pool` - a transaction pool that exists in the store of Helios -> Ethereum transactions waiting to be placed into a transaction batch.
- `Transaction batch` - A transaction batch is a set of Ethereum transactions (i.e. withdrawals) to be sent from the Hyperion Ethereum contract at the same time. Batching the transactions reduces the individual costs of processing the withdrawals on Ethereum. Batches have a maximum size (currently around 100 transactions) and are only involved in the Helios -> Ethereum flow. 
- `Hyperion Batch pool` - A transaction pool like structure that exists in the Helios Chain store, separate from the `Hyperion Tx pool`.  It stores transactions that have been placed in batches that are in the process of being signed or being submitted by the `Orchestrator Set`.
- `EthBlockConfirmationDelay` - An agreed upon number of Ethereum blocks confirmations that all oracle attestations are delayed by. No `Orchestrator` will attest to have seen an event occur on Ethereum until this number of blocks has elapsed as denoted by their trusted Ethereum full node. This prevents short forks/chain reorganizations from causing disagreements on the Helios Chain. The current value used is 12 block confirmations.
- `Observed` - Events on Ethereum are considered `Observed` when the `Eth Signers` of 66% of the active Helios validator set during a given block has submitted an oracle message attesting to seeing the event.
- `Validator set delta` - This is a term for the difference between the validator set currently in the Hyperion Ethereum contract and the actual validator set on the Helios Chain. Since the validator set may change every single block there is essentially guaranteed to be some nonzero `Validator set delta` at any given time.
- `Hyperion ID` - This is a random 32 byte value required to be included in all Hyperion signatures for a particular contract instance. It is passed into the contract constructor on Ethereum and used to prevent signature reuse when contracts may share a validator set or subsets of a validator set. 
- `Hyperion contract code hash` - This is the code hash of a known good version of the Hyperion contract solidity code. It will be used to verify exactly which version of the bridge will be deployed.
- `Voucher` - Represents a bridged ETH token on the Helios Chain side. Their denom is has a `hyperion` prefix and a hash that is build from contract address and contract token. The denom is considered unique within the system.
- `Counterpart` - to a `Voucher` is the locked ETH token in the contract
- `Delegate keys` - when an `Operator` sets up the `Eth Signer` and `Oracle` they assign `Delegate Keys` by sending a message containing these keys using their `Validator` address. There is one delegate Ethereum key, used for signing messages on Ethereum and representing this `Validator` on Ethereum and one delegate Helios Chain account key that is used to submit `Oracle` messages.
