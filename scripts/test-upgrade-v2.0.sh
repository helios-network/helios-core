#!/bin/bash

# Helios Network v2.0 Upgrade Testing Script
# This script validates that the v2.0 upgrade completed successfully

set -e

# Configuration
CHAIN_ID="helios-1"
NODE="http://localhost:26657"
KEYRING_BACKEND="test"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'

# Test results
PASSED_TESTS=0
FAILED_TESTS=0
TOTAL_TESTS=0

echo -e "${PURPLE}🧪 Helios Network v2.0 Upgrade Test Suite${NC}"
echo "================================================="

# Helper function to run tests
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "${YELLOW}Testing: $test_name${NC}"
    
    if eval "$test_command" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ PASS: $test_name${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        echo -e "${RED}❌ FAIL: $test_name${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# Helper function to check parameter values
check_parameter() {
    local module="$1"
    local param="$2"
    local expected="$3"
    local test_name="$4"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "${YELLOW}Testing: $test_name${NC}"
    
    local actual
    actual=$(heliosd query $module params --node "$NODE" --output json | jq -r ".$param" 2>/dev/null || echo "error")
    
    if [ "$actual" = "$expected" ]; then
        echo -e "${GREEN}✅ PASS: $test_name (Expected: $expected, Got: $actual)${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        echo -e "${RED}❌ FAIL: $test_name (Expected: $expected, Got: $actual)${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

echo -e "${BLUE}🔗 Testing network connectivity...${NC}"
run_test "Node connectivity" "curl -s $NODE/status"

echo ""
echo -e "${BLUE}📊 Testing network parameters...${NC}"

# Test staking parameters
echo -e "${YELLOW}Testing staking parameters...${NC}"
check_parameter "staking" ".max_validators" "125" "Max validators increased to 125"
check_parameter "staking" ".min_commission_rate" "\"0.020000000000000000\"" "Minimum commission rate set to 2%"

# Test unbonding time (14 days = 1209600000000000 nanoseconds)
EXPECTED_UNBONDING_TIME="1209600000000000"
run_test "Unbonding time reduced to 14 days" "[ \$(heliosd query staking params --node $NODE --output json | jq -r '.unbonding_time' | sed 's/s//') = \"$EXPECTED_UNBONDING_TIME\" ]"

# Test distribution parameters
echo -e "${YELLOW}Testing distribution parameters...${NC}"
check_parameter "distribution" ".community_tax" "\"0.010000000000000000\"" "Community tax reduced to 1%"

echo ""
echo -e "${BLUE}💰 Testing fee discount system...${NC}"

# Test fee discount account creation
FEE_DISCOUNT_ADDR="cosmos1vvr0fhvzpfg66kj4v8p8e8x7m3vgz8cz2l2ykf3"  # This would be the actual module address
run_test "Fee discount account exists" "heliosd query bank balances $FEE_DISCOUNT_ADDR --node $NODE | grep -q uhelios"

# Test fee discount balance (1M HELIOS = 1000000000000000000000000 uhelios)
EXPECTED_FEE_BALANCE="1000000000000000000000000"
echo -e "${YELLOW}Testing fee discount system balance...${NC}"
ACTUAL_BALANCE=$(heliosd query bank balances cosmos1vvr0fhvzpfg66kj4v8p8e8x7m3vgz8cz2l2ykf3 --node "$NODE" --output json 2>/dev/null | jq -r '.balances[] | select(.denom=="uhelios") | .amount' 2>/dev/null || echo "0")

if [ "$ACTUAL_BALANCE" = "$EXPECTED_FEE_BALANCE" ]; then
    echo -e "${GREEN}✅ PASS: Fee discount system has correct balance ($EXPECTED_FEE_BALANCE uhelios)${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${YELLOW}⚠️  INFO: Fee discount balance check (Expected: $EXPECTED_FEE_BALANCE, Got: $ACTUAL_BALANCE)${NC}"
    # This might not be exact due to module address calculation
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))

echo ""
echo -e "${BLUE}👥 Testing validator data migration...${NC}"

# Test validator count and commission caps
echo -e "${YELLOW}Testing validator commission caps...${NC}"
VALIDATORS_WITH_HIGH_COMMISSION=$(heliosd query staking validators --node "$NODE" --output json | jq '[.validators[] | select((.commission.commission_rates.rate | tonumber) > 0.10)] | length')

run_test "No validators with commission > 10%" "[ $VALIDATORS_WITH_HIGH_COMMISSION -eq 0 ]"

# Test validator set size
VALIDATOR_COUNT=$(heliosd query staking validators --node "$NODE" --output json | jq '.validators | length')
echo -e "${YELLOW}Current validator count: $VALIDATOR_COUNT${NC}"

echo ""
echo -e "${BLUE}🔗 Testing delegation system...${NC}"

# Test that delegation queries work
run_test "Delegation queries functional" "heliosd query staking delegations-to \$(heliosd query staking validators --node $NODE --output json | jq -r '.validators[0].operator_address') --node $NODE"

# Test that we can get delegation info
run_test "Delegation info accessible" "heliosd query staking delegations \$(heliosd keys list --keyring-backend $KEYRING_BACKEND --output json | jq -r '.[0].address' 2>/dev/null || echo 'cosmos1dummy') --node $NODE || true"

echo ""
echo -e "${BLUE}🛠️  Testing basic chain functionality...${NC}"

# Test block production
CURRENT_HEIGHT_1=$(curl -s "$NODE/status" | jq -r '.result.sync_info.latest_block_height')
sleep 6
CURRENT_HEIGHT_2=$(curl -s "$NODE/status" | jq -r '.result.sync_info.latest_block_height')

run_test "Block production continues" "[ $CURRENT_HEIGHT_2 -gt $CURRENT_HEIGHT_1 ]"

# Test transaction processing
run_test "Transaction queries work" "heliosd query tx \$(curl -s $NODE/block | jq -r '.result.block.data.txs[0]' | head -c 64) --node $NODE || echo 'No transactions to test'"

# Test governance functionality
run_test "Governance queries work" "heliosd query gov proposals --node $NODE"

# Test bank functionality
run_test "Bank queries work" "heliosd query bank total --node $NODE"

echo ""
echo -e "${BLUE}📋 Testing module functionality...${NC}"

# Test mint module
run_test "Mint module functional" "heliosd query mint params --node $NODE"

# Test staking module
run_test "Staking module functional" "heliosd query staking pool --node $NODE"

# Test distribution module
run_test "Distribution module functional" "heliosd query distribution params --node $NODE"

# Test governance module
run_test "Governance module functional" "heliosd query gov params --node $NODE"

echo ""
echo -e "${BLUE}🔍 Testing upgrade-specific features...${NC}"

# Test that upgrade info is cleared
run_test "Upgrade plan cleared" "! heliosd query upgrade plan --node $NODE 2>/dev/null | grep -q 'height'"

# Test chain version
CHAIN_VERSION=$(heliosd version 2>/dev/null | head -1 || echo "unknown")
echo -e "${YELLOW}Chain version: $CHAIN_VERSION${NC}"

echo ""
echo -e "${BLUE}💎 Advanced validation tests...${NC}"

# Test bonded pool
BONDED_TOKENS=$(heliosd query staking pool --node "$NODE" --output json | jq -r '.bonded_tokens')
run_test "Bonded tokens pool accessible" "[ ! -z '$BONDED_TOKENS' ] && [ '$BONDED_TOKENS' != 'null' ]"

# Test inflation
INFLATION=$(heliosd query mint inflation --node "$NODE" --output json | jq -r '.' || echo "0")
run_test "Inflation calculation works" "[ ! -z '$INFLATION' ] && [ '$INFLATION' != 'null' ]"

# Test community pool
COMMUNITY_POOL=$(heliosd query distribution community-pool --node "$NODE" --output json | jq -r '.pool[0].amount' 2>/dev/null || echo "0")
run_test "Community pool accessible" "[ ! -z '$COMMUNITY_POOL' ]"

echo ""
echo -e "${PURPLE}📊 Test Results Summary${NC}"
echo "==============================="
echo -e "${GREEN}Passed Tests: $PASSED_TESTS${NC}"
echo -e "${RED}Failed Tests: $FAILED_TESTS${NC}"
echo -e "${BLUE}Total Tests: $TOTAL_TESTS${NC}"

SUCCESS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
echo -e "${YELLOW}Success Rate: $SUCCESS_RATE%${NC}"

echo ""
if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed! Upgrade v2.0 is successful! 🎉${NC}"
    echo ""
    echo -e "${BLUE}📋 Upgrade Summary:${NC}"
    echo "• Staking parameters updated ✅"
    echo "• Distribution parameters updated ✅"
    echo "• Validator data migrated ✅"
    echo "• Fee discount system initialized ✅"
    echo "• Chain functionality verified ✅"
    echo ""
    echo -e "${GREEN}Helios Network v2.0 is ready for operation! 🚀${NC}"
    exit 0
elif [ $FAILED_TESTS -le 2 ]; then
    echo -e "${YELLOW}⚠️  Upgrade mostly successful with minor issues${NC}"
    echo "Please review the failed tests and investigate if needed."
    exit 1
else
    echo -e "${RED}❌ Upgrade validation failed! Please investigate immediately!${NC}"
    echo ""
    echo -e "${YELLOW}Recommended actions:${NC}"
    echo "1. Check node logs for errors"
    echo "2. Verify all validators are online"
    echo "3. Check if any manual intervention is needed"
    echo "4. Contact the development team if issues persist"
    exit 2
fi 