// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev The HyperionI contract's address.
address constant Hyperion_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000900;

/// @dev The HyperionI contract's instance.
HyperionI constant HYPERION_CONTRACT = HyperionI(Hyperion_PRECOMPILE_ADDRESS);

/// @author Helios Team
/// @title Hyperion Precompiled Contract
/// @dev The interface through which solidity contracts
/// @custom:address 0x0000000000000000000000000000000000000900
interface HyperionI {

    function addCounterpartyChainParams(
        uint64 hyperionId,
        string memory bridgeChainName,
        string memory contractSourceHash,
        string memory bridgeCounterpartyAddress,
        uint64 bridgeChainId,
        uint64 bridgeContractStartHeight
    ) external returns (bool success);

    function setOrchestratorAddresses(
        address orchestratorAddress,
        uint64 hyperionId
    ) external returns (bool success);

    /// @notice Requests data from a cross-chain source
    /// @param chainId The target chain ID
    /// @param contractAddress The source address on the target chain
    /// @param abiCall The ABI-encoded function call
    /// @param callbackSelector The selector of the callback function
    /// @param maxGasPrice Maximum gas price allowed for the callback
    /// @param gasLimit Maximum gas limit
    /// @return taskId A unique identifier for the data request
    function requestData(
        uint64 chainId,
        address contractAddress,
        bytes calldata abiCall,
        string memory callbackSelector,
        uint256 maxGasPrice,
        uint256 gasLimit
    ) external returns (uint256 taskId);

    function updateCounterpartyChainInfosParams(
        uint64 bridgeChainId,
        string memory bridgeChainLogo,
        string memory bridgeChainName
    ) external returns (bool success);

}
