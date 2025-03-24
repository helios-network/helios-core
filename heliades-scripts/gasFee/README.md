# Gas Fee Estimation

## Change direcoty to testing folder
```shell
cd heliades-scripts/gasFee
```

## Deploy counter contract

```shell
# optional
rm -rf deployments 
# deploy counter contract
npx hardhat deploy --network helios
```
## Create ERC20 token
```shell
node ./contracts/precompile/create.js
```

## Test gas consumption

```shell
npx hardhat test test/gas-analysis-helios.test.js --network helios 
```
# Test block base fee adjustment

```shell
npx hardhat test test/base-fee-adjustment.test.js --network helios
```

