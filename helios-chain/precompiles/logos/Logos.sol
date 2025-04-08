// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev The LogosI contract's address.
address constant Logos_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000901;

/// @dev The LogosI contract's instance.
LogosI constant LOGOS_CONTRACT = LogosI(Logos_PRECOMPILE_ADDRESS);

/// @author Helios Team
/// @title Logos Precompiled Contract
/// @dev The interface through which solidity contracts
/// @custom:address 0x0000000000000000000000000000000000000901
interface LogosI {

    event LogoUploaded(
        address indexed fromAddress,
        string hash
    );

    function uploadLogo(
        string memory logoBase64
    ) external returns (bool success);

}
