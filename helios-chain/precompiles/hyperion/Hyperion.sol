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
        string memory contractSourceHash,
        string memory bridgeCounterpartyAddress,
        uint64 bridgeChainId,
        uint64 bridgeContractStartHeight
    ) external returns (bool success);

    function setOrchestratorAddresses(
        address orchestratorAddress
    ) external returns (bool success);

}
