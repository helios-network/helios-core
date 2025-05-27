// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

import "../common/Types.sol";

/// @dev The IGov contract's address.
address constant GOV_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000805;

/// @dev The IGov contract's instance.
IGov constant GOV_CONTRACT = IGov(GOV_PRECOMPILE_ADDRESS);

/**
 * @dev VoteOption enumerates the valid vote options for a given governance proposal.
 */
enum VoteOption {
    // Unspecified defines a no-op vote option.
    Unspecified,
    // Yes defines a yes vote option.
    Yes,
    // Abstain defines an abstain vote option.
    Abstain,
    // No defines a no vote option.
    No,
    // NoWithWeto defines a no with veto vote option.
    NoWithWeto
}
/// @dev WeightedVote represents a vote on a governance proposal
struct WeightedVote {
    uint64 proposalId;
    address voter;
    WeightedVoteOption[] options;
    string metadata;
}

/// @dev WeightedVoteOption represents a weighted vote option
struct WeightedVoteOption {
    VoteOption option;
    string weight;
}

/// @dev DepositData represents information about a deposit on a proposal
struct DepositData {
    uint64 proposalId;
    address depositor;
    Coin[] amount;
}

/// @dev TallyResultData represents the tally result of a proposal
struct TallyResultData {
    string yes;
    string abstain;
    string no;
    string noWithVeto;
}

struct Asset {
    string denom;
    string contractAddress;
    string chainId;
    uint32 decimals;
    uint64 baseWeight;
    string metadata;
}

/// @dev WeightUpdateData represents the details for updating an asset's weight.
struct WeightUpdateData {
    string denom; // Asset denomination (e.g., USDT)
    string magnitude; // Magnitude of the update (e.g., "small", "medium", "high")
    string direction; // Direction of the update ("up" or "down")
}

/// @author The Evmos Core Team
/// @title Gov Precompile Contract
/// @dev The interface through which solidity contracts will interact with Gov
interface IGov {

    /// @dev Vote defines an Event emitted when a proposal voted.
    /// @param voter the address of the voter
    /// @param proposalId the proposal of id
    /// @param option the option for voter
    event Vote(address indexed voter, uint64 proposalId, uint8 option);

    /// @dev VoteWeighted defines an Event emitted when a proposal voted.
    /// @param voter the address of the voter
    /// @param proposalId the proposal of id
    /// @param options the options for voter
    event VoteWeighted(address indexed voter, uint64 proposalId, WeightedVoteOption[] options);

    /// TRANSACTIONS

    /// @dev vote defines a method to add a vote on a specific proposal.
    /// @param voter The address of the voter
    /// @param proposalId the proposal of id
    /// @param option the option for voter
    /// @param metadata the metadata for voter send
    /// @return success Whether the transaction was successful or not
    function vote(
        address voter,
        uint64 proposalId,
        VoteOption option,
        string memory metadata
    ) external returns (bool success);

    /// @dev voteWeighted defines a method to add a vote on a specific proposal.
    /// @param voter The address of the voter
    /// @param proposalId The proposal id
    /// @param options The options for voter
    /// @param metadata The metadata for voter send
    /// @return success Whether the transaction was successful or not
    function voteWeighted(
        address voter,
        uint64 proposalId,
        WeightedVoteOption[] calldata options,
        string memory metadata
    ) external returns (bool success);
     
    /// QUERIES

    /// @dev getVote returns the vote of a single voter for a
    /// given proposalId.
    /// @param proposalId The proposal id
    /// @param voter The voter on the proposal
    /// @return vote Voter's vote for the proposal
    function getVote(
        uint64 proposalId,
        address voter
    ) external view returns (WeightedVote memory vote);

    /// @dev getVotes Returns the votes for a specific proposal.
    /// @param proposalId The proposal id
    /// @param pagination The pagination options
    /// @return votes The votes for the proposal
    /// @return pageResponse The pagination information
    function getVotes(
        uint64 proposalId,
        PageRequest calldata pagination
    )
        external
        view
        returns (WeightedVote[] memory votes, PageResponse memory pageResponse);

    /// @dev getDeposit returns the deposit of a single depositor for a given proposalId.
    /// @param proposalId The proposal id
    /// @param depositor The address of the depositor
    /// @return deposit The deposit information
    function getDeposit(
        uint64 proposalId,
        address depositor
    ) external view returns (DepositData memory deposit);

    /// @dev getDeposits returns all deposits for a specific proposal.
    /// @param proposalId The proposal id
    /// @param pagination The pagination options
    /// @return deposits The deposits for the proposal
    /// @return pageResponse The pagination information
    function getDeposits(
        uint64 proposalId,
        PageRequest calldata pagination
    )
        external
        view
        returns (DepositData[] memory deposits, PageResponse memory pageResponse);

    /// @dev getTallyResult returns the tally result of a proposal.
    /// @param proposalId The proposal id
    /// @return tallyResult The tally result of the proposal
    function getTallyResult(
        uint64 proposalId
    ) external view returns (TallyResultData memory tallyResult);

    function addNewAssetProposal(
        string memory title,
        string memory description,
        Asset[] memory assets,
        uint256 initialDepositAmount 
    ) external returns (uint64 proposalId);

    /// @dev Propose to update an asset's weight in the consensus.
    /// @param title The title of the proposal.
    /// @param description A description of why the update is necessary.
    /// @param updates Array of weight updates to be applied.
    /// @return proposalId The unique ID of the proposal created.
    function updateAssetProposal(
        string memory title,
        string memory description,
        WeightUpdateData[] memory updates,
        uint256 initialDepositAmount
    ) external returns (uint64 proposalId);

    /// @dev Propose to remove assets from the consensus.
    /// @param title The title of the proposal.
    /// @param description A description of why the assets should be removed.
    /// @param denoms Array of asset denominations to be removed.
    /// @param initialDepositAmount Initial deposit amount required for the proposal.
    /// @return proposalId The unique ID of the proposal created.
    function removeAssetProposal(
        string memory title,
        string memory description,
        string[] memory denoms,
        uint256 initialDepositAmount
    ) external returns (uint64 proposalId);

    /**
     * @dev Submits a proposal to update consensus parameters.
     * @param title The title of the proposal
     * @param description The description of the proposal
     * @param maxGas The new maximum gas limit per block
     * @param maxBytes The new maximum block size
     * @param initialDepositAmount The initial deposit amount in ahelios
     * @return proposalId The ID of the created proposal
     */
    function updateBlockParamsProposal(
        string memory title,
        string memory description,
        int64 maxGas,
        int64 maxBytes,
        uint256 initialDepositAmount
    ) external payable returns (uint64 proposalId);

    /**
     * @dev Submits a proposal to update hyperion parameters.
     * @param title The title of the proposal
     * @param description The description of the proposal
     * @param msg The json message to be executed
     * @param initialDepositAmount The initial deposit amount in ahelios
     * @return proposalId The ID of the created proposal
     */
    function hyperionProposal(
        string memory title,
        string memory description,
        string memory msg,
        uint256 initialDepositAmount
    ) external payable returns (uint64 proposalId);
}
