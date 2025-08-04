#!/bin/bash

# Helios Network v2.0 Upgrade Proposal
# This script submits a governance proposal for the v2.0 major upgrade

set -e

# Configuration
CHAIN_ID="helios-1"
NODE="http://localhost:26657"
KEYRING_BACKEND="test"
FEES="20000000000000000uhelios"
GAS="auto"
GAS_ADJUSTMENT="1.5"

# Upgrade configuration
UPGRADE_NAME="v2.0"
UPGRADE_HEIGHT_OFFSET=2000  # More time for major upgrade preparation
UPGRADE_INFO='{"binaries":{"linux/amd64":"https://github.com/your-org/helios-core/releases/download/v2.0.0/helios-v2.0.0-linux-amd64.tar.gz","linux/arm64":"https://github.com/your-org/helios-core/releases/download/v2.0.0/helios-v2.0.0-linux-arm64.tar.gz","darwin/amd64":"https://github.com/your-org/helios-core/releases/download/v2.0.0/helios-v2.0.0-darwin-amd64.tar.gz","darwin/arm64":"https://github.com/your-org/helios-core/releases/download/v2.0.0/helios-v2.0.0-darwin-arm64.tar.gz","windows/amd64":"https://github.com/your-org/helios-core/releases/download/v2.0.0/helios-v2.0.0-windows-amd64.zip"}}'

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'

echo -e "${PURPLE}🚀 Helios Network v2.0 Major Upgrade Proposal${NC}"
echo "====================================================="

# Validate input
if [ -z "$1" ]; then
    echo -e "${RED}❌ Error: Please provide the proposer key name${NC}"
    echo "Usage: $0 <proposer-key-name>"
    echo "Example: $0 validator"
    exit 1
fi

PROPOSER_KEY="$1"

# Check node connectivity
echo -e "${YELLOW}🔗 Checking node connectivity...${NC}"
if ! curl -s "$NODE/status" > /dev/null; then
    echo -e "${RED}❌ Error: Cannot connect to node at $NODE${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Node connectivity verified${NC}"

# Get current height and calculate upgrade height
echo -e "${YELLOW}📏 Calculating upgrade height...${NC}"
CURRENT_HEIGHT=$(curl -s "$NODE/status" | jq -r '.result.sync_info.latest_block_height')
UPGRADE_HEIGHT=$((CURRENT_HEIGHT + UPGRADE_HEIGHT_OFFSET))

echo -e "${BLUE}Current Height: $CURRENT_HEIGHT${NC}"
echo -e "${BLUE}Upgrade Height: $UPGRADE_HEIGHT${NC}"
echo -e "${YELLOW}Estimated Time: $(date -d '+14 days')${NC}"

# Verify proposer account
echo -e "${YELLOW}👤 Verifying proposer account...${NC}"
if ! heliosd keys show "$PROPOSER_KEY" --keyring-backend "$KEYRING_BACKEND" > /dev/null 2>&1; then
    echo -e "${RED}❌ Error: Key '$PROPOSER_KEY' not found${NC}"
    exit 1
fi

PROPOSER_ADDRESS=$(heliosd keys show "$PROPOSER_KEY" --keyring-backend "$KEYRING_BACKEND" -a)
BALANCE=$(heliosd query bank balances "$PROPOSER_ADDRESS" --node "$NODE" --output json | jq -r '.balances[] | select(.denom=="uhelios") | .amount')

echo -e "${BLUE}Proposer: $PROPOSER_ADDRESS${NC}"
echo -e "${BLUE}Balance: $BALANCE uhelios${NC}"

# Check sufficient balance for major upgrade proposal
MIN_BALANCE=15000020000000000000000  # 15M HELIOS deposit + fees for major upgrade
if [ "$BALANCE" -lt "$MIN_BALANCE" ]; then
    echo -e "${RED}❌ Error: Insufficient balance. Need at least 15M HELIOS for major upgrade proposal${NC}"
    exit 1
fi

# Create comprehensive proposal
PROPOSAL_FILE="/tmp/upgrade-v2.0-proposal.json"

cat > "$PROPOSAL_FILE" << EOF
{
  "messages": [
    {
      "@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
      "authority": "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
      "plan": {
        "name": "$UPGRADE_NAME",
        "height": "$UPGRADE_HEIGHT",
        "info": "$UPGRADE_INFO"
      }
    }
  ],
  "metadata": "Helios Network v2.0 - Major Network Enhancement with Enhanced Delegation Logic, Fee Optimization, and Network Parameter Improvements",
  "deposit": "15000000000000000000000uhelios",
  "title": "Helios Network v2.0 Major Upgrade",
  "summary": "This proposal initiates the Helios Network major upgrade to v2.0, introducing enhanced delegation logic, fee optimization systems, and significant network improvements."
}
EOF

# Create detailed description
DESCRIPTION="# Helios Network v2.0 Major Upgrade

## 🌟 Executive Summary
This governance proposal requests approval for the Helios Network major upgrade to version 2.0, scheduled for block height $UPGRADE_HEIGHT. This is a comprehensive upgrade that enhances core functionality, improves user experience, and strengthens network economics.

## 🚀 Major Features & Improvements

### 🔗 Enhanced Delegation System
- **Delegation Timestamp Tracking**: New system to track delegation history
- **Loyalty Rewards**: Automatic bonuses for long-term delegators (1% bonus for 1000+ HELIOS delegations)
- **Improved Undelegation Logic**: More efficient processing of undelegation requests

### 💰 Fee Optimization System
- **Fee Discount Program**: 1M HELIOS allocated for loyal delegator fee discounts
- **Dynamic Fee Structure**: More predictable and user-friendly fee calculation
- **Reduced Community Tax**: From 2% to 1% for better validator economics

### ⚙️ Network Parameter Improvements
- **Unbonding Time**: Reduced from 21 days to 14 days for better UX
- **Max Validators**: Increased to 125 for better decentralization
- **Commission Caps**: Automatic capping of validator commissions above 10%
- **Minimum Commission**: Set to 2% for sustainable validator operations

### 🛡️ Validator Enhancements
- **Metadata Migration**: Automatic migration of validator data to new format
- **Commission Rate Optimization**: Automatic adjustment of excessive commission rates
- **Performance Tracking**: Enhanced validator performance monitoring

## 📊 Technical Specifications

### Migration Process
1. **Pre-Migration Validation**: Comprehensive checks before upgrade execution
2. **Delegation Logic Migration**: Process all existing delegations
3. **Parameter Updates**: Apply new network parameters
4. **Fee System Initialization**: Deploy fee discount system
5. **Validator Data Migration**: Update validator metadata
6. **Post-Migration Validation**: Verify all changes applied correctly

### Data Migration
- **Total Delegations**: All existing delegations will be processed
- **Validator Data**: Commission rates and metadata updated
- **New Accounts**: Fee discount system account created
- **Parameter Changes**: All network parameters updated atomically

### Security Measures
- **Atomic Operations**: All changes applied atomically during upgrade
- **Rollback Protection**: Comprehensive validation prevents incomplete upgrades
- **Data Integrity**: All existing balances and states preserved
- **Backward Compatibility**: No breaking changes to core functionality

## 💼 Economic Impact

### Benefits for Delegators
- **Reduced Unbonding Time**: Faster access to undelegated funds (14 vs 21 days)
- **Loyalty Rewards**: Bonus rewards for long-term participation
- **Fee Discounts**: Reduced transaction costs for active participants
- **Better Validator Choice**: More validators available (125 vs current)

### Benefits for Validators
- **Lower Community Tax**: More rewards stay with validators (1% vs 2%)
- **Fair Commission Caps**: Protection against excessive commission rates
- **Improved Decentralization**: More validator slots available
- **Enhanced Metadata**: Better validator information system

### Network Benefits
- **Increased Decentralization**: More validator positions
- **Improved Economics**: Better balance between all participants
- **Enhanced UX**: Faster undelegation and better fee structure
- **Long-term Sustainability**: Incentives for long-term participation

## 📅 Implementation Timeline

- **Proposal Submission**: $(date)
- **Voting Period**: 7 days from submission
- **Preparation Period**: 7 days after passing (if approved)
- **Upgrade Execution**: Block $UPGRADE_HEIGHT
- **Network Resume**: Immediately after successful upgrade

## 🧪 Testing & Validation

### Testnet Testing
- **Complete upgrade simulation** on dedicated testnet
- **Migration testing** with production-like data
- **Performance benchmarking** before and after upgrade
- **Validator coordination** testing and procedures

### Pre-Upgrade Checklist
- [ ] Binary release and checksum verification
- [ ] Validator communication and coordination
- [ ] Backup procedures verified
- [ ] Cosmovisor configuration updated
- [ ] Emergency procedures documented

## 🚨 Risk Assessment

### Technical Risks
- **Migration Complexity**: Extensive testing completed to minimize risks
- **Network Halt Duration**: Expected halt time < 30 minutes
- **Data Migration**: All processes extensively tested on testnet

### Mitigation Strategies
- **Comprehensive Testing**: Full testnet simulation completed
- **Validator Coordination**: Direct communication with all validators
- **Emergency Procedures**: Rollback procedures documented
- **Community Support**: 24/7 technical support during upgrade

## 🗳️ Voting Options

- **YES**: Approve the v2.0 upgrade at block $UPGRADE_HEIGHT
- **NO**: Reject the upgrade proposal
- **NO WITH VETO**: Reject and penalize proposal deposit
- **ABSTAIN**: Do not participate in the decision

## 📱 Community Resources

- **Documentation**: Full upgrade guide available
- **Support**: Community Discord and Telegram channels
- **Technical Details**: GitHub repository with complete changes
- **Validator Guide**: Detailed upgrade procedures for validators

---

**Upgrade Height**: $UPGRADE_HEIGHT
**Estimated Date**: $(date -d '+14 days')
**Binary Release**: Available 48 hours before upgrade
**Community Vote Required**: >50% YES votes with >33.4% participation

This upgrade represents a major step forward for the Helios Network, improving user experience, validator economics, and network security. We encourage all stakeholders to review the changes and participate in the governance process."

echo "$DESCRIPTION" > "/tmp/upgrade-v2.0-description.md"

echo -e "${YELLOW}📋 Proposal Summary:${NC}"
echo "Title: Helios Network v2.0 Major Upgrade"
echo "Upgrade Height: $UPGRADE_HEIGHT"
echo "Deposit: 15,000,000 HELIOS"
echo "Estimated Date: $(date -d '+14 days')"

# Confirmation
echo -e "${PURPLE}⚠️  MAJOR UPGRADE PROPOSAL${NC}"
echo -e "${YELLOW}This is a major network upgrade with significant changes.${NC}"
echo "Key changes:"
echo "• Enhanced delegation system with loyalty rewards"
echo "• Fee discount system (1M HELIOS allocation)"
echo "• Reduced unbonding time (14 days)"
echo "• Increased validator set (125 validators)"
echo "• Commission rate optimizations"
echo "• Reduced community tax (1%)"
echo ""
read -p "Do you want to submit this major upgrade proposal? (y/N): " -n 1 -r
echo

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}❌ Proposal submission cancelled${NC}"
    rm -f "$PROPOSAL_FILE" "/tmp/upgrade-v2.0-description.md"
    exit 0
fi

# Submit proposal
echo -e "${YELLOW}📤 Submitting v2.0 major upgrade proposal...${NC}"

TX_HASH=$(heliosd tx gov submit-proposal "$PROPOSAL_FILE" \
    --from "$PROPOSER_KEY" \
    --keyring-backend "$KEYRING_BACKEND" \
    --chain-id "$CHAIN_ID" \
    --node "$NODE" \
    --fees "$FEES" \
    --gas "$GAS" \
    --gas-adjustment "$GAS_ADJUSTMENT" \
    --yes \
    --output json | jq -r '.txhash')

if [ "$TX_HASH" = "null" ] || [ -z "$TX_HASH" ]; then
    echo -e "${RED}❌ Error: Failed to submit proposal${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Major upgrade proposal submitted successfully!${NC}"
echo -e "${BLUE}Transaction Hash: $TX_HASH${NC}"

# Wait and get proposal ID
echo -e "${YELLOW}⏳ Waiting for transaction confirmation...${NC}"
sleep 8

PROPOSAL_ID=$(heliosd query tx "$TX_HASH" --node "$NODE" --output json | jq -r '.events[] | select(.type=="submit_proposal") | .attributes[] | select(.key=="proposal_id") | .value' | tr -d '"')

if [ "$PROPOSAL_ID" = "null" ] || [ -z "$PROPOSAL_ID" ]; then
    echo -e "${YELLOW}⚠️  Could not retrieve proposal ID automatically${NC}"
    echo "Please check transaction: heliosd query tx $TX_HASH"
else
    echo -e "${GREEN}✅ Proposal ID: $PROPOSAL_ID${NC}"
    
    echo -e "${YELLOW}📋 Proposal Status:${NC}"
    heliosd query gov proposal "$PROPOSAL_ID" --node "$NODE"
    
    echo ""
    echo -e "${BLUE}🗳️  Next Steps for Validators:${NC}"
    echo "1. Review the upgrade documentation"
    echo "2. Test the upgrade on testnet"
    echo "3. Vote on the proposal:"
    echo "   heliosd tx gov vote $PROPOSAL_ID yes --from <validator-key> --chain-id $CHAIN_ID --fees $FEES"
    echo ""
    echo -e "${BLUE}📊 Monitoring:${NC}"
    echo "• Proposal status: heliosd query gov proposal $PROPOSAL_ID --node $NODE"
    echo "• Voting progress: heliosd query gov votes $PROPOSAL_ID --node $NODE"
    echo "• Current tally: heliosd query gov tally $PROPOSAL_ID --node $NODE"
fi

# Cleanup
rm -f "$PROPOSAL_FILE" "/tmp/upgrade-v2.0-description.md"

echo ""
echo -e "${PURPLE}🎉 Helios Network v2.0 Upgrade Proposal Submitted!${NC}"
echo -e "${YELLOW}⏰ Important Dates:${NC}"
echo "• Current Height: $CURRENT_HEIGHT"
echo "• Upgrade Height: $UPGRADE_HEIGHT"
echo "• Estimated Upgrade: $(date -d '+14 days')"
echo "• Voting Ends: $(date -d '+7 days')"
echo ""
echo -e "${GREEN}Thank you for participating in Helios Network governance! 🚀${NC}" 