// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.18;

import "../ERC20Creator.sol";

contract Erc20CreatorCaller {
    function callCreateErc20(
        string memory name,
        string memory symbol,
        uint256 totalSupply,
        uint8 decimals
    ) external returns (address) {
        return
            ERC20CREATOR_CONTRACT.createErc20(
                name,
                symbol,
                totalSupply,
                decimals
            );
    }
}
