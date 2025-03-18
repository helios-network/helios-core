// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Counter {
    uint256 public count;
    
    event CountIncremented(uint256 newCount);

    constructor() {
        count = 0;
    }
    
    function increment() external returns (uint256) {
        count += 1;
        emit CountIncremented(count);
        return count;
    }
    
    function getCount() external view returns (uint256) {
        return count;
    }
}