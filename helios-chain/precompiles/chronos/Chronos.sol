// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev The ChronosI contract's address.
address constant Chronos_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000830;

/// @dev The ChronosI contract's instance.
ChronosI constant CHRONOS_CONTRACT = ChronosI(Chronos_PRECOMPILE_ADDRESS);

/// @author Helios Team
/// @title Chronos Precompiled Contract
/// @dev The interface through which solidity contracts can create cron tasks
/// @custom:address 0x0000000000000000000000000000000000000830
interface ChronosI {

    event EVMCallScheduled(
        uint64 scheduleId
    );

    function createCron(
        address contractAddress,
        string memory abi,
        string memory methodName,
        string[] memory params,
        uint64 frequency,
        uint64 expirationBlock,
        uint64 gasLimit,
        uint64 maxGasPrice
    ) external returns (bool success);

    function updateCron(
        uint64 cronId,
        uint64 newFrequency,
        string[] memory newParams,
        uint64 newExpirationBlock,
        uint64 newGasLimit,
        uint64 newMaxGasPrice
    ) external returns (bool success);

    function cancelCron(
        uint64 cronId
    ) external returns (bool success);

}
