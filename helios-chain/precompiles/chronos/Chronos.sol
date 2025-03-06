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

    function scheduleEVMCall(
        address addr,
        address contractAddress,
        string memory abi
    ) external returns (bool success);

}
