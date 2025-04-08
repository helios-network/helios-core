# Testing the Helios Revenue and Auto-Registration Features

This directory contains scripts and instructions for testing two key features:
1. Fee Distribution (Revenue Module)
2. Automatic Contract Registration

## Prerequisites

- Node.js
- Install dependencies
```sh
npm install
```
- Helios local chain running
```sh
make full-install
```

# Test 1: Manual Revenue Registration and Distribution

## 1. Deploying the counter contract for revenue testing

```bash
node deploy-contract-revenue.js
```

- Expected output:
```sh

=== Deployment Script Started ===

Deployer Information:
  EVM Address (hex): 0x17267eB1FEC301848d4B5140eDDCFC48945427Ab
  Helios Address (bech32): helios1zun8av07cvqcfr2t29qwmh8ufz29gfatfue0cf
  Balance: 99.9998 HELIOS

Deploying Counter Contract...
  Deployment Nonce: 1
  Transaction hash: 0x33e80d3eda446557e264186bd84defc820bc655b98ae2b3678e25713084542c0
  Waiting for deployment confirmation...

Counter Contract Deployed!
  Contract Address: 0xB4bB7B6037DE7E8Ac7CcDEFE927ea13e94ff99d9

=== Deployment Script Completed ===
```

## 2. Add a mnemonic to the key ring

```sh
heliades keys add mykey --recover --keyring-backend=test
```
When prompted, enter the mnemonic:
```sh
web tail earth lesson domain feel slush bring amused repair lounge salt series stock fog remind ripple peace unknown sauce adjust blossom atom hotel
```
- expected output:
```sh
- address: helios1zun8av07cvqcfr2t29qwmh8ufz29gfatfue0cf
  addressEthereum: 0x17267eB1FEC301848d4B5140eDDCFC48945427Ab
  name: mykey
  pubkey: '{"@type":"/helios.crypto.v1beta1.ethsecp256k1.PubKey","key":"A+4ebHYgWYaDHGk6Ji8xp0b38a/gCCUcjHJQIzdl1ksi"}'
  type: local
```

## 3. Register the counter contract for fee distribution

- IMPORTANT: make sure to use the correct contract address, deployer address and nonce used during contract deployment.
- The contract address is `0xB4bB7B6037DE7E8Ac7CcDEFE927ea13e94ff99d9` and the deployer address is `helios1zun8av07cvqcfr2t29qwmh8ufz29gfatfue0cf` and the nonce is `1`.
  
```sh
heliades tx revenue register 0xB4bB7B6037DE7E8Ac7CcDEFE927ea13e94ff99d9 1 helios1zun8av07cvqcfr2t29qwmh8ufz29gfatfue0cf \
--from mykey \
--keyring-backend=test \
--chain-id=4242 \
--fees 250000000000000ahelios \
--gas auto \
-y
```

  
- expected output:
```sh
gas estimate: 115623
code: 0
codespace: ""
data: ""
events: []
gas_used: "0"
gas_wanted: "0"
height: "0"
info: ""
logs: []
raw_log: ""
timestamp: ""
tx: null
txhash: 292EFB882F5279C25C774E94109502A7ABE485B922EE260450794631DDDB1D63
```

## 4. Check the contract is registered

```sh
heliades query revenue contract 0xB4bB7B6037DE7E8Ac7CcDEFE927ea13e94ff99d9
```

- expected output:
```sh
revenue:
  contract_address: 0xB4bB7B6037DE7E8Ac7CcDEFE927ea13e94ff99d9
  deployer_address: helios1zun8av07cvqcfr2t29qwmh8ufz29gfatfue0cf
  withdrawer_address: ""
```

## 5. Run the revenue distribution test script

```sh
node test-revenue.js
```

- expected output:
```sh
Transferring 20 HELIOS to the user...
Direct WebSocket connected!
Websocket connected and subscribed to transactions events

=== New Transaction Detected ===
Transferred 20 HELIOS to the user!

=== Initial State ===
Deployer Address: 0x17267eB1FEC301848d4B5140eDDCFC48945427Ab
User Address: 0x1dC48AD55caef5543585494c38f2FDbd806910F9
Contract Address: 0xB4bB7B6037DE7E8Ac7CcDEFE927ea13e94ff99d9

Initial Balances:
Deployer: 79.97427879 HELIOS
User: 20.0 HELIOS

=== Executing Transactions ===
Starting nonce: 0

Sending transaction 1/3...
Using nonce: 0
Transaction hash: 0xbc902f0702c27502cbf18c948f480ee6146ea333bb0f166f3220056805cd5aea
Waiting for transaction to be mined...

=== New Transaction Detected ===

🎉 Found revenue distribution events!

Event type: distribute_dev_revenue
  sender: 0x1dC48AD55caef5543585494c38f2FDbd806910F9
  contract: 0xB4bB7B6037DE7E8Ac7CcDEFE927ea13e94ff99d9
  withdrawer_address: helios1zun8av07cvqcfr2t29qwmh8ufz29gfatfue0cf
  amount: 2525000000000000
  msg_index: 0
Transaction mined in block: 242

Sending transaction 2/3...
Using nonce: 1
Transaction hash: 0x6e59347656c5e666a29aed5919128613561a095afd0b4cf5bf23676cecbc6370
Waiting for transaction to be mined...

=== New Transaction Detected ===

🎉 Found revenue distribution events!

Event type: distribute_dev_revenue
  sender: 0x1dC48AD55caef5543585494c38f2FDbd806910F9
  contract: 0xB4bB7B6037DE7E8Ac7CcDEFE927ea13e94ff99d9
  withdrawer_address: helios1zun8av07cvqcfr2t29qwmh8ufz29gfatfue0cf
  amount: 2525000000000000
  msg_index: 0
Transaction mined in block: 243

Sending transaction 3/3...
Using nonce: 2
Transaction hash: 0x3c3821c1fab7eeaf0ad20da33e341267d84521b565d30876a1026335dfe8df44
Waiting for transaction to be mined...

=== New Transaction Detected ===

🎉 Found revenue distribution events!

Event type: distribute_dev_revenue
  sender: 0x1dC48AD55caef5543585494c38f2FDbd806910F9
  contract: 0xB4bB7B6037DE7E8Ac7CcDEFE927ea13e94ff99d9
  withdrawer_address: helios1zun8av07cvqcfr2t29qwmh8ufz29gfatfue0cf
  amount: 2525000000000000
  msg_index: 0
Transaction mined in block: 245

=== Results ===
Total Gas Used: 750000
Total Fees Paid: 0.07575 HELIOS

Balance Changes:
Deployer:
  Initial: 79.97427879 HELIOS
  Final: 79.98185379 HELIOS
  Change: 0.007575 HELIOS

User:
  Initial: 20.0 HELIOS
  Final: 19.92425 HELIOS
  Change: -0.07575 HELIOS

Expected Revenue (10% of fees): 0.007575 HELIOS
Actual Revenue: 0.007575 HELIOS
Percentage of Fees Received: 10 %
```

# Test 2: Automatic Contract Registration

## 1. Deploying a contract to test auto-registration

```bash
node deploy-contract-auto.js
```

## 2. Verify automatic registration

```sh
heliades query revenue contract <CONTRACT_ADDRESS>
```

- Expected output:
```sh
revenue:
  contract_address: <CONTRACT_ADDRESS>
  deployer_address: <DEPLOYER_ADDRESS>
  withdrawer_address: ""
```

## 3. Run the auto-registration test script

```bash
node test-auto-registration.js
```

This script will:
1. Deploy a new contract
2. Verify it was automatically registered
3. Execute transactions to generate fees
4. Verify revenue distribution is working

For detailed events and transaction monitoring, check the logs in the terminal.