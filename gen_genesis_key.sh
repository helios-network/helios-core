#!/bin/bash

rm -rf ./helios_keys_backup

# Generate 11 keys
keys=(
    "validator_key"
    "user1_key"
    "user2_key"
    "user3_key"
    "user4_key"
    "ocr_admin_key"
    "signer1_key"
    "signer2_key"
    "signer3_key"
    "signer4_key"
    "signer5_key"
)

# Ensure secure permissions
mkdir -p "./helios_keys_backup"
umask 077

# Generate keys
for keyname in "${keys[@]}"; do
    # Generate key and capture output using test backend
    output=$(heliades keys add "$keyname" --keyring-backend test --output json)
    
    # Extract mnemonic and address
    mnemonic=$(echo "$output" | jq -r '.mnemonic')
    address=$(echo "$output" | jq -r '.address')
    
    # Save mnemonic to a secure file
    echo "$mnemonic" > "./helios_keys_backup/${keyname}_mnemonic.txt"
    
    # Print key details
    #echo "Key Name: $keyname"
    #echo "Address: $address"
    #echo "Mnemonic saved to: ./helios_keys_backup/${keyname}_mnemonic.txt"
    #echo "---"
    echo "\"$mnemonic\""
done
