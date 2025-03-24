// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

contract GasConsumer {
    // A function that consumes a variable amount of gas
    function consumeGas(uint256 gasToConsume) external {
        uint256 startGas = gasleft();
        uint256 targetGas = startGas - gasToConsume;
        
        // Consume gas by performing computations in a loop
        bytes32 hash = bytes32(0);
        while (gasleft() > targetGas) {
            hash = keccak256(abi.encodePacked(hash, block.number));
        }
    }
    
    // Alternative implementation using storage writes which consume more gas
    uint256[] private data;
    
    function consumeGasWithStorage(uint256 iterations) external {
        for (uint256 i = 0; i < iterations; i++) {
            data.push(i);
        }
    }
} 