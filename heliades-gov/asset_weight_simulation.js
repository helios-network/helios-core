import readline from 'readline';

// Asset configuration example
const assets = [
  { name: "ETH", marketPrice: 3000 },
  { name: "USDT", marketPrice: 1 },
  { name: "BNB", marketPrice: 500 },
  { name: "POL", marketPrice: 0.7 },
];

// Function to calculate weights
function calculateWeights(assets) {
  const totalPrice = assets.reduce((sum, asset) => sum + asset.marketPrice, 0);
  return assets.map(asset => ({
    name: asset.name,
    weight: (asset.marketPrice / totalPrice).toFixed(4),
  }));
}

// Function to simulate supply distribution
function calculateSupplyDistribution(totalSupply, weights) {
  return weights.map(asset => ({
    name: asset.name,
    supply: (totalSupply * asset.weight).toFixed(2),
  }));
}

// Function to calculate yearly inflation rewards
function calculateInflationRewards(supply, inflationRate) {
  return (supply * (inflationRate / 100)).toFixed(2);
}

// Function to calculate daily rewards for each asset
function calculateDailyRewards(totalInflationRewards, weights) {
  const dailyRewards = totalInflationRewards / 365;
  return weights.map(asset => ({
    name: asset.name,
    dailyReward: (dailyRewards * asset.weight).toFixed(4),
  }));
}

// Main simulation function
function runSimulation(totalSupply, inflationRate) {
  const weights = calculateWeights(assets);
  console.log("\nAsset Weights:");
  weights.forEach(asset =>
    console.log(`${asset.name}: ${(asset.weight * 100).toFixed(2)}%`)
  );

  const supplyDistribution = calculateSupplyDistribution(totalSupply, weights);
  console.log("\nSupply Distribution:");
  supplyDistribution.forEach(asset =>
    console.log(`${asset.name}: ${asset.supply}`)
  );

  const totalInflationRewards = calculateInflationRewards(totalSupply, inflationRate);
  console.log("\nTotal Inflation Rewards (Yearly):", totalInflationRewards, "HELIOS");

  const dailyRewards = calculateDailyRewards(totalInflationRewards, weights);
  console.log("\nDaily Rewards per Asset:");
  dailyRewards.forEach(asset =>
    console.log(`${asset.name}: ${asset.dailyReward} HELIOS`)
  );

  return { weights, supplyDistribution, totalInflationRewards, dailyRewards };
}

// CLI for user input
const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout,
});

console.log("=== Asset Weight Simulation ===");
rl.question("Enter total supply (HELIOS): ", totalSupply => {
  rl.question("Enter inflation rate (%): ", inflationRate => {
    runSimulation(parseFloat(totalSupply), parseFloat(inflationRate));
    rl.close();
  });
});
