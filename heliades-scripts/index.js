const ethers = require('ethers');
const WebSocket = require('ws');
const fs = require('fs');

// const RPC_URL = 'https://testnet1.helioschainlabs.org';
const RPC_URL = 'http://localhost:8545';
const COSMOS_RPC_WS = 'ws://localhost:26657/websocket'; // WebSocket Cosmos RPC

const PRIVATE_KEY = '2c37c3d09d7a1c957f01ad200cec69bc287d0a9cc85b4dce694611a4c9c24036';

const PRIVATE_KEY2 = 'e1ab51c450698b0af4722e074e39394bd99822f0b00f1a787a131b48c14d4483'

const PRECOMPILE_CONTRACT_ADDRESS = '0x0000000000000000000000000000000000000806';

const provider = new ethers.JsonRpcProvider(RPC_URL);

const wallet = new ethers.Wallet(PRIVATE_KEY, provider);
const wallet2 = new ethers.Wallet(PRIVATE_KEY2, provider);


const abi = [
  {
    "inputs": [
      { "internalType": "string", "name": "name", "type": "string" },
      { "internalType": "string", "name": "symbol", "type": "string" },
      { "internalType": "string", "name": "denom", "type": "string" },
      { "internalType": "uint256", "name": "totalSupply", "type": "uint256" },
      { "internalType": "uint8", "name": "decimals", "type": "uint8" },
      { "internalType": "string", "name": "logoBase64", "type": "string" }
    ],
    "name": "createErc20",
    "outputs": [
      { "internalType": "address", "name": "tokenAddress", "type": "address" }
    ],
    "stateMutability": "nonpayable",
    "type": "function"
  }
];

const delegateAbi = [
  {
    "inputs": [
      { "internalType": "address", "name": "delegatorAddress", "type": "address" },
      { "internalType": "address", "name": "validatorAddress", "type": "address" },
      { "internalType": "uint256", "name": "amount", "type": "uint256" },
      { "internalType": "string", "name": "denom", "type": "string" }
    ],
    "name": "delegate",
    "outputs": [{ "internalType": "bool", "name": "success", "type": "bool" }],
    "stateMutability": "nonpayable",
    "type": "function"
  }
];

const unstakingAbi = [
  {
    "inputs": [
      { "internalType": "address", "name": "delegatorAddress", "type": "address" },
      { "internalType": "address", "name": "validatorAddress", "type": "address" },
      { "internalType": "uint256", "name": "amount", "type": "uint256" },
      { "internalType": "string", "name": "denom", "type": "string" }
    ],
    "name": "undelegate",
    "outputs": [{ "internalType": "bool", "name": "success", "type": "bool" }],
    "stateMutability": "nonpayable",
    "type": "function"
  }
];

const proposalAbi = [
  {
    "inputs": [
      { "internalType": "string", "name": "title", "type": "string" },
      { "internalType": "string", "name": "description", "type": "string" },
      {
        "components": [
          { "internalType": "string", "name": "denom", "type": "string" },
          { "internalType": "string", "name": "contractAddress", "type": "string" },
          { "internalType": "string", "name": "chainId", "type": "string" },
          { "internalType": "uint32", "name": "decimals", "type": "uint32" },
          { "internalType": "uint64", "name": "baseWeight", "type": "uint64" },
          { "internalType": "string", "name": "metadata", "type": "string" }
        ],
        "internalType": "struct Asset[]",
        "name": "assets",
        "type": "tuple[]"
      },
      {
        "internalType": "uint256",
        "name": "initialDepositAmount",
        "type": "uint256"
      }
    ],
    "name": "addNewAssetProposal",
    "outputs": [
      { "internalType": "uint64", "name": "proposalId", "type": "uint64" }
    ],
    "stateMutability": "nonpayable",
    "type": "function"
  }
];

const updateProposalAbi = [
  {
    "inputs": [
      {
        "internalType": "string",
        "name": "title",
        "type": "string"
      },
      {
        "internalType": "string",
        "name": "description",
        "type": "string"
      },
      {
        "components": [
          { "internalType": "string", "name": "denom", "type": "string" },
          { "internalType": "string", "name": "magnitude", "type": "string" },
          { "internalType": "string", "name": "direction", "type": "string" }
        ],
        "internalType": "struct WeightUpdateData[]",
        "name": "updates",
        "type": "tuple[]"
      },
      {
        "internalType": "uint256",
        "name": "initialDepositAmount",
        "type": "uint256"
      }
    ],
    "name": "updateAssetProposal",
    "outputs": [
      {
        "internalType": "uint64",
        "name": "proposalId",
        "type": "uint64"
      }
    ],
    "stateMutability": "nonpayable",
    "type": "function"
  }
];

const withdrawDelegatorRewardsAbi = [
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "delegatorAddress",
        "type": "address"
      },
      {
        "internalType": "address",
        "name": "validatorAddress",
        "type": "address"
      }
    ],
    "name": "withdrawDelegatorRewards",
    "outputs": [
      {
        "components": [
          {
            "internalType": "string",
            "name": "denom",
            "type": "string"
          },
          {
            "internalType": "uint256",
            "name": "amount",
            "type": "uint256"
          }
        ],
        "internalType": "struct Coin[]",
        "name": "amount",
        "type": "tuple[]"
      }
    ],
    "stateMutability": "nonpayable",
    "type": "function"
  }
]


const contract = new ethers.Contract(PRECOMPILE_CONTRACT_ADDRESS, abi, wallet);


const tokenName = 'BNB';
const tokenSymbol = 'BNB';
const tokenDenom = 'uBNB'; // denomination of one unit of the token
const tokenTotalSupply = ethers.parseUnits('100', 18);
const tokenDecimals = 18;

async function create(){
  try {
    console.log('Création du token ERC20...');
    
    const tx = await contract.createErc20(tokenName, tokenSymbol, tokenDenom, tokenTotalSupply, tokenDecimals, "iVBORw0KGgoAAAANSUhEUgAAAMgAAADICAYAAACtWK6eAAAAAXNSR0IB2cksfwAAAAlwSFlzAAALEwAACxMBAJqcGAAAHZ5JREFUeJztXQmUnUWVBgMcMR1IAiaAWzSGhFFHBWcSFCZoBpdhZgDnOKOoxCWOM6OCozMEgowJqOi4EY4CDsGTEB3XcUZcSG9JOjsJS9JJSCCdjaydpNP1ln77ezX3Vv2v08vrfv9S9df9/1ffOZ/HE7r/d+vW/frVX3Xr3rPOslAK1jx2DHA8cArwSuAtwLuBjwKfAK4HdgIPALuBSWAOWAZWHJadf0s6P3PA+Z31zjMec56Jz36H81n4mWNMj9/CYhAgKCcBrwHeBlwG3A48DEw7wc41Ez+jD3jE+ezHgbcDrwVONu0fiwYBBNvZwInAmcAFwHbgsRAEEJTHHVvxG+dqZwxnm/anRQzQ++TLURiXA+cBW4FHneWP6aD3y7Ij6jbgZ4DTk0+OM+1miyiBybX8bGepcohAUOsmLgeXA6/DsZv2vwVBnG5uwuXTLOBiJ2Ci/C0R5NsFx/4g8OpeuwxrbDC52zRtgChMByg1VsWCS0y7O9YogMk+BziHyRfXNIFApE7cIVvp+Owc0/NnoQkwuZcC5zO5q2M66KJK9N2dwMtMz6eFIsBkXgJcCDxJIMDiwlPARehb0/Nr4RMweVOBDwPzBAIqrkTfPoK+Nj3fFi7B5EEYfmMkCARQoxB9fS/63vT8W4wAJs8v7mLypdJ0wDQq0feYZTDBdDxYOIDJeBnweuBeFk7uk+XorDhz8V6cG9Px0bBgK0Ru1AzgRisMksQ52QS8wh46hgwmzzLuATICgWA5OpkzV+eajpvYg7WMRXHMBu4kMPGW3vg88Lpk8ytMh1E8Ac49j8mDviyBybb0R5w7PGg8z3Q8xQa9zU0ojiuAuwhMsKUa7hJzCisCi4AAR97K5Mmt6Um1VEuc07mm4yuyAOdNAC4lMJGWeonXlO25iRcwmVi4lcDkWYZDnOtLTccdeSRXjUNx3AzsITBpluHyNM59us2+l9QEk0UR8H0jR2CyLM0Q534usweLgwEOORe4hDXmVVfLwcQYwFiwB4sIcMT5TBZCMz0xlrT4Y+D5puPTKMABFwI3E5gMS5rE2LjQdJwaAQz8ImAHgUmwpE2MkYtMx2uoYLJcZxcB51tGgxgrk0zHbShgUhxbCDjdMlrEmIm3SJhcVtlvDku/xNiJ53ILBnYBs+8cozKx6vU8sfLVxu0gToyheL24M7mVa3erRmCifTLP7f8+r5QywBTP7f0GT7S90rhdhImxFI8tYCYPAe05xwhMP/tBXk7v4kNRSm7j6advMG4fYWJMRfswkcmi0EsIOJMcUxuu5sWeVcOEMRTFk808tf4dxu0lSoytaKalOImHmFtl00cGMNE2mWe7vgZLqb664qgCl13ZF++xy67hxNi6NdXcZDrcvYPJrFybeFhl60Se2T6PVwo9roUxTCj5E7xv263wrAnmx0OHGGM3m453T2DyPsdpAs4jweS6t/NS7waI8LJvcQz4OuHF0x08ufYtxsdFiHg9Ihr3SZi8CWgvOzXLbdvCsV9CUBeDC2OYUIq8cOSn8BmvMz5OIsSYo38zkckrlKadZZYt43hm5xd4JXdEvTCGoJw9xDM7/gk+d5z5cZvnUtPxPyLAOORcAk4yyCaefvpveSm1Q7swhkJsC2/5K2GDeT8Y5VzWQvClncnSPA1bfSS5eirPH14KS59C6OLoR7nA8y8tEUs70/4wSIzBGab1MAhMFnVryLpVifZLefbFr+Jax5wwhgBP5LMv3A22XWLcP4aIsUijOJ1TDnQ+AaeETFhObb6el9MvYEia1kQNVMQJffqpObxBl13zExTKnDJZK7eByoE28eTat/Li6TWcpjCGosKLPaudbeGGEgrG5GzT4sA8q8YpJN06kef2fpNXisx01HtGpdDLc13fEGMw7sfwiLFppjNv4klRpuceAk7Qz5YLeWbHZ3lZ+7YtHiTq/VYqZw/zzPbPiDEZ92s4vAd7yYQuECab18S+P0dq/VUiYVAGr8bABfFltn+WZ56/XaSU6EWZF088yZPrrjLu3xCIMRrurhaTbc82Ehi8NiZw2/al/xKpHTqByzVctg1c+mBSYm7/A/DfUlo/G8eWf+lHYqym/a2ZGKvhtYNjsv9cPNuetVzA+7Z9nFdyx/UGJ748n14rXvhrvzw3iTT3Uu9TXPeyq5I7xvu2fkyM3bj/9RBj9fqwxIG5VnsJDFqxMMbx1KZ317y8pDgceblvD09vft8IwhgulPTTf83LmX2a7eJi7OgD9IXx+VBPjFn9uVpMtlqO1bdHYuWreP7Icv2HfeUcz+65lyfaL/Nh46thKXY/PCOv2cas8AX6xPS8KCbG7F26xTGRxawPOf4lr+RP6Q06kXm7nCdXTwtsb7JjOi8c/bmeDOGBJhdO8fSW9xufH8XE2J2oUyCLCAxSLWHdnd78flj27NUSaOI0+5kPql22wLP6nv0QPPtFPTbDck6II57vJIt0iWMqMEFggJqEciHPbP9H8ZdTBfAlP7vrS3qDDJ6d3fXv8A3YrcZmGDtuNbOW8ebnQx8xhqfqEMgjBAanncmOGbAc+gn3vXNUKfHC8f8V2b2h2QxLt0L3EwG2pCvi8hWO3bT/Q+LDqsVxCTBPYGChMbXxXd7yrSplXmJP89T6PzNn84aZvJR4hrs/1JRbzamN1xj3d8jEWL5EpUDi9+7hhq0TYAnzZVjCnBw9zIqMZ3Z+jidaLzJuc6L1YrDl82BTYnSbYTmV3fVvjVwAYqEqcVwGPElgQK4plkkn/sALx37NE6vfEDzo2ibx/MEfwgomPeRLI8tz+77ja9t2mM1r3gxLs/9RttRBm3L7vitsHGQzjCF/8CExpuB+ng5Lu98Lu8NcUioixnTwQg/wkDsJDMYd4UU7+8KCQX/x8ZQ48/wXFbwsNw1YwnBe6t3IU5uuC24z/AXP7LxdZNsO/Osu7pkreFlObZotK6qgzYlnYQyzeOC0d/Tz7ruG+DmETQn1vDOoOM4BHicwkDoTdgHve+4jo27Viu3WZxVst+KJu4pKhxBk6Wdu5OXsSyPbnNnP01tuUBJ0wmYFY+977sPg565R/Lwb/Px3UTmNx9j2nw4PvzyHwCBGZXLNFbCU+pW7HRxxYPcTWBpcbtRmTA4sdP/O3f31cgGWML8xfs9c+vmXHvz8U+N+dsk5fsUxBriSwABqBxmmXuz7T3+pF+Ucz3V9LfR0CrwjnjvwoK+UFrxnjhXgsRJ86H7e+80Afv469bSVduAYPwKZBkwTGMAQjuPpp/9GLD+CoSKWZGGVypEn3y8EtFkuFftwCaPdz+qSJKWfbwjFzz6IMT7Nj0AeJGD8oAkT6d9Mdfp3RbxwJ9ddqWUCUxvfyYs9HQrtlcB75vhsHX5Orr8KfLKJq/fzJvFsgkJZ7FUceJ32MAHDJVvGywtEJX0XiCrFpNiyVTZ5aPOeez1VcfdsMzw7t+c+hakhTXJrGHyhzWb0MywViV35PdTb3OT+Wi78wtUEjO4nplLoRYUXT7Xx1IY/V2v3urepK1o9zOSyeDZ+hkqbcSu7eGol131Ji+C5yazILq90CkSck+z4nL4Evf62B6fV2Vzo4ZnOT+urUAK+ELWFFSVB1gJBgbhbZsEPjmeUlleaBIKpGLl93w4tPUT0Iez6eqAlF/5utus+0YQnFJvbLpa9EzUsuQgKBGN+vBuBzGbEukKpFQi8LJ5eL3p2mHhZ9PvSjsseeQoetv+bRGUX1XfjCQoEY75+oTn4oeUEjB1ENQLxehdcL91u+5ZTO3n6mZuN26v6bjxBgSAfryOOl6NASC2vkIEFUs7y7J5FapIKO6aJpVlyzZsCP2u0g0OxQ4WJkAqWU9Lm76qxWRwcfivw3XiiAjl0qn2U7tLwA9MJGDl8gv0KpFLg+cPL4PffqCCYL4N3iPvE6fCZZy8VARP42QNTTyDwCsd/raRzFNqcf+mRQTajUBIrXxt8TryknkRHIMjLRxPIPAIGDp8MHwLBtG5cxgTvvARLiy0fcBL0hq/By5kDvK/zk1xFhiwmL+L9eBXJiX1bPzpiUiEmSCq5J4/Ji9s+4St1hrBA5o0kDjwcbCNg4DD6EkjuWMAcoCZxui5Txeu/nJbYFhKtBjAFHzMD3EBF+jsGup8tbMICaWW1eq8zWdLnKAEDa0xCuALBW3m+TpQxOe/AYiPJefiZuf3f8/zXHN918gcf9v2uE0OBoAaGlwaCf5zFiG3vnpmEkAQiDsg+H7jsKAoLi1DjOYJ2YbRdLA/16lyxrWtzoYdnd8/3fPgYQ4GgBmbWEsjdBIwzJhD1KRZlZwkzU5tfvBdpqIeKWFLiLcQGFghyQS2BtBMwLHSB4A6XaL6pq71BpcSL3b9VspPWb3PHNHjm7zRWnq+IS2jJjisaVSDttQRC9mqtToEUDi/z/Gw/qOSPiwopgXaosFDc7ju05kgNBN4KbFCBHBsqjskEjDIjkKM/8/zsIBB3tr1usYq74H/Py316So2OBPwWaVCBICcNFMi1BAwama0TnEMp9wWbqQpEGlctZl1/2SVKGIVQrLoWsGyScoHg2I/+Igr1uK4ZKJDbCRhUfzLWvV3u8bu4X0FaIFWIFJiFotf6ULsC3QVXBCwWoUwg4v7KeuX3VzTytoECeZyAQe4o7ld8pu6kREIg0lKRtIg91+WBXfXOvf6GOfWANYZVCETeX/lU1DrsLhPi6JXVS7YTMMgTRaLf3vtFtY+akxIZgVRREWnwJbaZU+m/Xjj+f4EEguk+omFQSPdXFLOTYbUTJi9IHSFgkC+KItM9a4ZPTuQEQg+F7t/6FkjxVLvWM6AQKC9Qwf9MYTHoGiV2eQY0lLECCQ5sqeBVIPL+yk3G40EBsRzQFBTIVSwmPQfl/YofiJdfK5DgwBR8twI5c38leFFsIkRNXIkCuYWAMUqJk5Y/9OOau0NWIO6Bldvr+S+xagr4+lEl91cI8hbSOVjB2MTdpHFbgYyM4on6AnHr54hyAQpkCQFDjNEKZGQUT/zR+PwY5hIUyO8IGGIFQhBWIGOfQIGsJ2DIqEysfI3oQKvjoMkKZGQUTz6pfj6dg97EquD34kPgOhRIJwFDRiW+dMtUhWqRaSuQMFA8uULtPA5IFcIEVNNx5YKdKJADBAypL5AqMNnt2K+UNZQRfcY1FsWOLLD/4t77lfgY52posmlEBLIfBdJNwBD3AnFQyR0R12ODt/pqEq2bKaV4mEY5tUPeKlRQ9QS7/5ZhroYiIgLpRoEkCRjiWSBVlGAyMbkv+FYjnSRBU8BAxh6EKhqeoi9Lye0jflZEBJJEgeQIGOJbIAJYxA0PBlW0fMY08677jaaZhw5Y+uQP/EDJshWfkT/0WN3+ixERSA4FQrKSiSeBVOe5nOHZF78iUk4Cf2bH5bxw5L+NXFQKDfCyjPlWybVvDS4M8Hn2hbtHzK4eiogIpHQWASOUCcSZddny+an38ODVDseJlsb4vLhBVITcNldN//in3g3vLegj9+9wERFIJYYCkQheWXGgUC7gmee/KJ4ZdVQKp+Bb9j84a1XTMAh97McvURJIbJZYgwJBCCR4YelBwQDvOHhPW1+5HY2A5VTxZLMoOq3UJ+DjGAukFI+X9FrxoEEg/faImr3u7sabR4WXktt4auM1WnwRc4HkIr/NO2JY5I5rE4hgf+/BHs+2hQU8AM3s+hJPtL1Smx+kQLyXao2IQJKRPSisGxy6BVINkLbJom8g3r8mAzwF3784nPGvfE2cBdIdvVQTl8BKhjh59Z6NPQNVJEHK2r7tnu1UjVLvJrGrFHgHr3UCjOmd7gSSj61A9kcnWdEj3AqkcOwX8Af3sOwDqKC5PTbBwVSNsIHNcvq2fiy4v0Ujn5vAJ4eEb+r9fMwF0hmJdHd/Aul2LRD5C0VePPEHNe3a2iaJ/iLhHDKWef7QEjX9F7EV3Inf99vtXiDeawVHRCAi3f0JAoZoEkj9Owf9AumPtyzPH/yhq/vso9v8RoizXs92e0Y5x5Nr3xzI1oHFLgbCnUBeG2eBPBGJK7e+BeLiUs4wgVTjru9F3vfcP8ReIEPLJQ2EK4GsirVAlkSiaIMJgVRRPL3W1xkCdYGMVHBvINwL5IRnsyMikAWRKPvjTyAnlAhEPKuUEYWkvSRBUhVIvZKtA2EFIsv+kC8c518g9Ws1uRFI/zMLva7vxpMTiMui3wPhTiCvi6tA+gvHYenRNAGDyAtEPrjMs3sWRU4g2GbBa2pMgwukv/Qo+eLV/gRyUo9AAPn934+cQNBmr3AvkJOenx0BgfQXrybf/sC/QKZYgWgXyJS4CqSTtYwdU22gs4yAQVYgfmAFoovLItOCzQpkFFiB6OKgFmykm3j6EkjhlBVIWAIBX3tFBAQyqInnJAIGaRBI/SodViAjw51AXh9XgZxpA+2I5DgBo8IXyJHlnp/dMAIB3zSoQI6dNRTwj+0EDFMokB5XAsFAzh9eyjEr1i3iL5Ay+GSZqyBOCoF4v1VJXCDttQRCNifLr0CSq90XQjtz4al+6Zr4CqTiNN+c5WFu3hBHgSyoJZBZjGiFE/8C8VhpsWU8z+z4l7pFCOIoEBwz1tFFH3ibm9gJBDUws5ZAJgKPEjDQnECqbL2I5/Z+CwKc1Xx2nARSKSZ4bt+3eQLG7G9uYicQ1MDEWgI5G9hGwEBFAjntXyCCTTy57q28eBpTwgcvu+IhkIpI5U+uexsPcn9dCsR9AmQVhAXSCv44e5hAHJHMI2CgEoHg7bhM5yfVVCrf/Je8nN7Nq0KJtkCwNOsLMKbrAwlDEHzb1/mJYTcR3YCwQObVFIcjkOkEDFQjEBELJVEJMbnmTwLbIIszf0Xco4iqQGRx73sCXycW4wOfYiMjv1UmCQvk8hEFcrr1fBTJIQJGqhFIFZWCKKLg5o56XaGser3z1zdaAklvfq+a9gYrXwO+/E7d9gb1QFQgh061nz+iPqrfIssJGKpWINVYyhx0yvsE7UrlxmZaAglMrHSP5YDAhypAVCCPj64OKZDZjNh2ryqBVFFKPOvs9QftStUIAmkSviolnlFqNkGBYMzPdiMQvEB1mIDB2gSCwFKh+YMPi9KhViC1ib7JH3xIS1lVggKRF6TcAH5wMQGDtQqkCtzDz+6er7wHe6QFAr7I7r5Da2FuggJZ7EocjkCuJmBwKAKRqPBS7wbZ2bXBBYI+QF/o7vhLUCCzXAukd4U4NKSzzGqdIJLn/Oy3e0NF9PNOrnmTApsnilJBuLWqDeW8s0wM3t4Ax1w4+kuuvRU2Vq6EucQ5NR5XZ3iIrRjhcHCUb5EHCRh+hi3j5F+3EApDi/I+O29TMolYdE6exqsFvjSnN7+Pq6jintn5BV+n4V6hrP+6erpfXg0QyDRGsRwQnt4+9+GazemVT2j2JbG1qaLqe9+zH3JO4wPalDnAM52fCi4MUcX9RjFG3RD917d+REFWgxZijE/zIxCsdkL2jggeeomCy7orqFcKouK5kqrvWNVw/wOuWyUPAi6nsBe8gqY4OBZZxT3YYV9dKOy/rpEY42M8C8QRyRwCAxh9stf+qej1rb1foKj6/pCSNA0MmMLx34igdxNkxZMtPNkxQ4FALxWV67W/yynsvx4C5/gShyOQc4DHCAxidOKya9tcsfzQjXLfHmftH9zm9JYPwPP2jfxZuDSBpZnXOxq1iKkmaLt2/yjrvx4KMbbP8S0QRyTzCQzEZdCNFz3A/dyR9gI3yYpebBY1cweUzsF7KNldX+YJhWczoqGPRvT3X1cg5hB5RyBxOAK5FHiSwGBcM7lmBixLVmhbdikVSNXmjuli+xOXXsm1b1H+fG0CEf3XVyjvvx4CMaYvDSwQRyQLCQzIM7EHBvYIV723r0MguqleINX+6+8yPjafXKhEHI5ALgHmCQzKM/EQLfP8v8KyJaksNBpdIOhL9KnO/uuaibE8WZlAHJE8QmBgvplY+SqxxapiB6dhBSL6rz8QSv91zXxEqTgcgUwFJggMLhBTm67jpd6N2gWSWP0Gntnxz3p3c+DZ+BkJF3fvgwoEfZba9G7j86eAGMNTlQvEEckiAgNUwr6tHxW9xXUJBHOb8C9uObVL/SUtcXnpZnj28+Iz3OSO+RWI7L/+UePzpZALtYjDEQiWBuojMEgllAdnD3lednkRiACeKB9+XMlpfLJjmiyXWs0g0CWQck4mQirov06IGLvDS/ooFsldjHhPQ89Bh6kX3e5TLzwLpD/osjz74kJfp/EYqLk994nAHfpMpQJRmFpDjBizd2kVhyOQCcC9BAaslh6S93wLREag6EvuvuxOk3Pi3sVrblcrFIhMzrxRSXImQXYxtzcGFYjkehazb5F+ivTvz42a/h1MIGeEgmnwmEtWWyhYuO7K+peXFAgEx5rZ+XlqdzRUEmP1+lDEIQTSMvZl8IEbCQxcG0Xdp6M/qxmcagTiBGeRib7lA6/8JlovFgHt6uwmkEAqMMafK6kbRpwb2YqxLwtNIEIkzWNnABmBwWulvPC0dpBQVArkTJwfFjlZ4vJS7rjr3/MnEFl2NLXxWuP+DYEYozNCFYcQiLyWew8BB+gnLrt2fak/oVCHQGTclrnntBiPAsExZGEsMV5ODeVXgN6u0yoTiUyH30nACaEQUytyBxbzXNd9dX/Wl0D8wKVAsnsWCdsjnB7ihxibwdLZFYhkNjBLwBkhsQmCbFLdn6MmECkMfYXyCBJjsn4huBAEEq07IyGRmkAakPMTK8aalocEGHMecBcBp5ChFYhRYiyeZ1oXg8DkrtYpAs4hQSsQY8SLUOHvWtUDk0utWwk4iAStQIzx1kTzK0zLYWSAgUsJOMk4MQV9tIIMyvSR2ecq3b1BuNR0/NcFk7laWwk4yzixYU/hyE/01O7CpEJ4dmJV8KZAMeFzwAmm498VmCz00EPAaSSIRRhE+VGf7coGC6PEiz0d4TTNiQ4x1tQUYAgLYPBNwBwB59Fg6wRRLypISaJK/gQ84+ONdAruhhhjN5mOd89Ito2rvrST6lZlmnjIiKfaXprSVEop2XyzsU7B3RBj69bUH5tMh7s/MNl7fQkBR5JjasNMXjzVXlccWHMqtf4dxu0lykeZqTwrVYABnAt8jIAzSbL/bvkQYM2p9NM3GLePMDGmzjUd30oAAzkfuJmAU0kSewHm9n9PVH3H5VSu6xt2OTU6MZbq9GuOGGBAFwBXE3AuWSZWTYlDzSndxBi6wHQ8awEM7CLgHgJOtowm8V75RabjWCtggJOYXW5ZeucW4CTT8RsKYKCvZPabxNI98ZujMcRRBZPLLftOYlmPHSzuy6qRwOSLu11uWY5EjI0LTcepUTC5BWzPSSyHEmMiXlu5fsHkYSKeitq0FEuMAcy+iMchoCowmZbycWYTHBuZOPeYvxft9BFdcBIcMQvYpso3HnHOb45s4mGYYPI+yXMEJs0yHOIFu2jd5zANJm8m2uu78SfOcTRuAlIEk2vSSLWhtnRFrIAz13R8RR69zU3VkkK27lZ8iHM5A+fWQhGYLE53B2uoMqexI84dVuGkVdQtLmArxlZrATdMwewYEedsdqKFcM2quIDJqvJY1r6XwMRbjk7sz4FtMsxWWW80OP1J8N1kA4trO7hoE+dko5ijFfbgzxhgArAdHPZM7LJCIUGcg73OnITb9sxiZMBkjAfeCUwTCJJGJfYhxzbh9lyDKmByJgIXAhMEAqZRiL5ehL43Pf8WLgGTNRX4MDBPIIDiygLwR+hr0/Nt4RMweZOBX2X2NF4l0Zf4LX2J6fm1UAQmEyDxkOoYgQCLKo85PrSJhXEFk2co7wG2M/tC74Zpx1dzmD3LaBzAZI8BTgM+ADxEIBCp8TBwseOjMabny8IgWHMTHjrOcsSCgdGIV3/LA0Qxq9fe6rOoBSbPU/4C+DhrjG8WHONyJnPcxpv2v0WEkJKp9pcDPw1sAR5l0f52KTtjaAPOA07v+aMtHGKhAEwWlcBDyJnABU6QRWFH7DiTL9lo8yxnDHb5ZBEOmKw1fA3wNuAyYCeTa3nc/QkjN6zifNYR4HbHBrTlWuBk0/6xsBgEJnfH8D1mCvBK4C3OX2+s+/UEcJ0jov3AbmCSyXI3ZSfYK87/zzn/rdv52U7nd/EZjznPxGdf5XzWeNZid5tU4/8B/5GVDEqnSNAAAAAASUVORK5CYII=");
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    receipt.logs.forEach(log => {
      try {
        const parsedLog = contract.interface.parseLog(log);
        if (parsedLog.name === 'ERC20Created') {
          console.log('Token créé avec succès :');
          console.log('Créateur :', parsedLog.args.creator);
          console.log('Adresse du token :', parsedLog.args.tokenAddress);
          console.log('Nom du token :', parsedLog.args.name);
          console.log('Symbole du token :', parsedLog.args.symbol);
        }
      } catch (err) {
      }
    });
  } catch (error) {
    console.error('Une erreur est survenue :', error);
  }
}

async function fetch(){
  const contractAddress = '0x80b5a32E4F032B2a058b4F29EC95EEfEEB87aDcd';

  const abi = [
    'function name() view returns (string)',
    'function symbol() view returns (string)',
    'function decimals() view returns (uint8)',
    'function totalSupply() view returns (uint256)',
    'function balanceOf(address account) view returns (uint256)',
  ];

  // Créer une instance du contrat
  const contract = new ethers.Contract(contractAddress, abi, provider);
  
  // Récupérer les informations du token
  const name = await contract.name();
  const symbol = await contract.symbol();
  const decimals = await contract.decimals();
  const totalSupply = await contract.totalSupply();
  const balance = await contract.balanceOf(wallet.address); // Assuming you want the balance of the wallet

  // Afficher les informations
  console.log('Token Name:', name);
  console.log('Token Symbol:', symbol);
  console.log('Token Decimals:', decimals);
  console.log('Total Supply:', totalSupply.toString());
  console.log('Balance of Wallet:', balance.toString());
}

async function delegate() {

  const validatorAddress = '0x17267eb1fec301848d4b5140eddcfc48945427ab'; // Adresse du validateur
  const delegateAmount = ethers.parseUnits('1.5', 18); // Montant à déléguer (ex: 10 tokens)
    // Extraire et afficher la clé publique
    console.log("wallet : ", wallet.address)
  try {
    console.log('Délégation en cours...');

    const contract = new ethers.Contract('0x0000000000000000000000000000000000000800', delegateAbi, wallet);
    const tx = await contract.delegate(wallet.address, validatorAddress, delegateAmount, "WETH");
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log('Délégation réussie !');
  } catch (error) {
    console.error('Erreur lors de la délégation :', error);
  }
}

async function addNewConsensusProposal() {
  const contract = new ethers.Contract("0x0000000000000000000000000000000000000805", proposalAbi, wallet2);

  const title = 'Whitelist WETH into the consensus with a base stake of power 100';
  const description = 'Explaining why WETH would be a good potential for Helios consensus and why it would secure the market';
  const assets = [
    {
      denom: 'WETH',
      contractAddress: '0x80b5a32E4F032B2a058b4F29EC95EEfEEB87aDcd', // Exact match to ABI
      chainId: 'ethereum',                                          // Exact match to ABI
      decimals: 6,
      baseWeight: 100,
      metadata: 'WETH stablecoin'
    }
  ];

  try {
    console.log('Ajout d\'une nouvelle proposition au consensus...');
    console.log('Arguments envoyés au contrat :', { title, description, assets });
    
    const tx = await contract.addNewAssetProposal(title, description, assets, "1000000000000000000", {
      gasPrice: 50000000000,
      gasLimit: 500000
    });
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log('Proposition soumise avec succès !');
  } catch (error) {
    console.error('Erreur lors de la soumission de la proposition :', error);
  }
}

async function updateConsensusProposal() {
  const contract = new ethers.Contract("0x0000000000000000000000000000000000000805", updateProposalAbi, wallet2);

  const title = 'Update WETH to higt';
  const description = 'update WETH';
  const updates = [
    {
      denom: 'BNB',
      magnitude: 'high',
      direction: 'up'
    }
  ];

  try {
    console.log('update proposition au consensus...');
    console.log('Arguments envoyés au contrat :', { title, description, updates });
    
    const tx = await contract.updateAssetProposal(title, description, updates, "1000000000000000000");
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log('Proposition soumise avec succès !');
  } catch (error) {
    console.error('Erreur lors de la soumission de la proposition :', error);
  }
}

const voteAbi = [
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "voter",
        "type": "address"
      },
      {
        "internalType": "uint64",
        "name": "proposalId",
        "type": "uint64"
      },
      {
        "internalType": "enum VoteOption",
        "name": "option",
        "type": "uint8"
      },
      {
        "internalType": "string",
        "name": "metadata",
        "type": "string"
      }
    ],
    "name": "vote",
    "outputs": [
      {
        "internalType": "bool",
        "name": "success",
        "type": "bool"
      }
    ],
    "stateMutability": "nonpayable",
    "type": "function"
  },
];

async function vote(){
  const contract = new ethers.Contract("0x0000000000000000000000000000000000000805", voteAbi, wallet);

  try {
    console.log('Ajout d\'une nouvelle proposition au consensus...');
    
    const tx = await contract.vote(wallet.address, 2, 1, "voting testtest");
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log('vote fait a success soumise avec succès !');
  } catch (error) {
    console.error('Erreur lors de la soumission de la proposition :', error);
  }
}

async function undelegate() {

  const validatorAddress = '0x17267eb1fec301848d4b5140eddcfc48945427ab'; // Adresse du validateur
  const delegateAmount = ethers.parseUnits('0.1', 18); // Montant à déléguer (ex: 10 tokens)
    // Extraire et afficher la clé publique
    console.log("wallet : ", wallet.address)
  try {
    console.log('UnDélégation en cours...');

    const contract = new ethers.Contract('0x0000000000000000000000000000000000000800', unstakingAbi, wallet);
    const tx = await contract.undelegate(wallet.address, validatorAddress, delegateAmount, "BNB");
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log('UnDélégation réussie !');
  } catch (error) {
    console.error('Erreur lors de la délégation :', error);
  }
}

async function getRewards() {

  const validatorAddress = '0x17267eb1fec301848d4b5140eddcfc48945427ab'; // Adresse Cosmos du validateur
  const delegateAmount = ethers.parseUnits('10', 18); // Montant à déléguer (ex: 10 tokens)
    // Extraire et afficher la clé publique
    console.log("wallet : ", wallet.address)
  try {
    console.log('WithdrawRewards en cours...');

    const contract = new ethers.Contract('0x0000000000000000000000000000000000000801', withdrawDelegatorRewardsAbi, wallet);
    const tx = await contract.withdrawDelegatorRewards.staticCall(wallet.address, validatorAddress, {
      gasLimit: 500000,
      gasPrice: ethers.parseUnits('20', "gwei")
    });
    console.log(tx);
    // console.log('Transaction envoyée, hash :', tx.hash);

    // const receipt = await tx.wait();
    // console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    // console.log('WithdrawRewards réussie !');
  } catch (error) {
    console.error('Erreur lors de la WithdrawRewards :', error);
  }
}

async function getRewards() {

  const validatorAddress = '0x17267eb1fec301848d4b5140eddcfc48945427ab'; // Adresse Cosmos du validateur
  const delegateAmount = ethers.parseUnits('10', 18); // Montant à déléguer (ex: 10 tokens)
    // Extraire et afficher la clé publique
    console.log("wallet : ", wallet.address)
  try {
    console.log('WithdrawRewards en cours...');

    const contract = new ethers.Contract('0x0000000000000000000000000000000000000801', withdrawDelegatorRewardsAbi, wallet);
    const tx = await contract.withdrawDelegatorRewards(wallet.address, validatorAddress);
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log('WithdrawRewards réussie !');
  } catch (error) {
    console.error('Erreur lors de la WithdrawRewards :', error);
  }
}

async function addCounterpartyChainParams() {
  console.log("wallet : ", wallet.address)
  try {
    console.log('addCounterpartyChainParams en cours...');

    const hyperionAbi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/hyperion/abi.json').toString()).abi;
    const contract = new ethers.Contract('0x0000000000000000000000000000000000000900', hyperionAbi, wallet);
    const tx = await contract.addCounterpartyChainParams(
      21, // New HyperionId
      "", // bridge contract hash
      "0x17AAd46782B95fbDE4ff1051397F24e554f6A487", // bridge contract address
      80002, // chainId
      19437427 // start height
    );
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log(receipt);
  } catch (error) {
    console.error('Erreur lors de la addCounterpartyChainParams :', error);
  }
}

async function setOrchestratorAddresses() {
  console.log("wallet : ", wallet.address)
  try {
    console.log('setOrchestratorAddresses en cours...');

    const chronosAbi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/hyperion/abi.json').toString()).abi;
    const contract = new ethers.Contract('0x0000000000000000000000000000000000000900', chronosAbi, wallet);
    const tx = await contract.setOrchestratorAddresses( // I'm validator and
      "0x17267eB1FEC301848d4B5140eDDCFC48945427Ab", // address of my hyperion Validator
      21 // HyperionId
    );
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log(receipt);
  } catch (error) {
    console.error('Erreur lors de la setOrchestratorAddresses :', error);
  }
}

async function sendToChain(amount) {
  console.log("wallet : ", wallet.address)
  try {
    console.log('sendToChain en cours...');

    const hyperionAbi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/hyperion/abi.json').toString()).abi;
    const contract = new ethers.Contract('0x0000000000000000000000000000000000000900', hyperionAbi, wallet);
    const tx = await contract.sendToChain( // I'm validator and
      80002, // chainId
      "0x17267eB1FEC301848d4B5140eDDCFC48945427Ab", // receiver
      "0xd4949664cd82660aae99bedc034a0dea8a0bd517", // token address (ex: USDT)
      ethers.parseEther(amount), // amount to transfer (ex: 10 USDT)
      ethers.parseEther("1"), // fee you want to pay (ex: 1 USDT)
      {
        gasLimit: 500000,
        gasPrice: ethers.parseUnits('20', "gwei")
      }
    );
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);
    console.log(receipt);
  } catch (error) {
    console.error('Erreur lors de la sendToChain :', error);
  }
}

async function createCron() {
  console.log("wallet : ", wallet.address)
  try {
    console.log('createCron en cours...');

    // let x = await wallet.sendTransaction({
    //   value: ethers.parseEther("500"),
    //   to: wallet2.address
    // });
    // await x.wait();

    const chronosAbi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/chronos/abi.json').toString()).abi;
    const contract = new ethers.Contract('0x0000000000000000000000000000000000000830', chronosAbi, wallet);
    const tx = await contract.createCron(
      "0xa80799d619abafB179e843025571d7913bC00ce8",
      `[ { "inputs": [], "name": "increment", "outputs": [], "stateMutability": "nonpayable", "type": "function" } ]`,
      "increment", // methodName
      [], // params
      1, // frequency
      0, // expirationBlock
      400000, // gasLimit
      ethers.parseUnits("2", "gwei"), // maxGasPrice
      ethers.parseEther("1")
    );
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log(receipt);
  } catch (error) {
    console.error('Erreur lors de la createCron :', error);
  }
}

async function cancelCron() {
  console.log("wallet : ", wallet.address)
  try {
    console.log('createCron en cours...');

    const chronosAbi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/chronos/abi.json').toString()).abi;
    const contract = new ethers.Contract('0x0000000000000000000000000000000000000830', chronosAbi, wallet);
    const tx = await contract.cancelCron(
      1
    );
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log(receipt);
  } catch (error) {
    console.error('Erreur lors de la cancelCron :', error);
  }
}

async function getEventsCronCreated() {
  const chronosAbi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/chronos/abi.json').toString()).abi;
  const wsProvider = new ethers.WebSocketProvider('ws://testnet1.helioschainlabs.org:8546');
  const contract = new ethers.Contract('0x0000000000000000000000000000000000000830', chronosAbi, wsProvider);

  contract.on('CronCreated', (from, to, cronId, event) => {
    console.log('New event received!');
    console.log('even:', event);
    console.log('cronId:', cronId.toString());
  });
}

async function getEventsCronCancelled() {
  const chronosAbi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/chronos/abi.json').toString()).abi;
  const wsProvider = new ethers.WebSocketProvider('ws://localhost:8546');
  const contract = new ethers.Contract('0x0000000000000000000000000000000000000830', chronosAbi, wsProvider);

  contract.on('CronCancelled', (from, to, cronId, event) => {
    console.log('New event received!');
    console.log('even:', event);
    console.log('cronId:', cronId.toString());
  });
}

async function getEvents() {
  const abiContract = [
    {
      "anonymous": false,
      "inputs": [
        {
          "indexed": false,
          "internalType": "uint256",
          "name": "newCount",
          "type": "uint256"
        }
      ],
      "name": "CountIncremented",
      "type": "event"
    }
  ];

  const wsProvider = new ethers.WebSocketProvider('ws://localhost:8546');
  const contract = new ethers.Contract('0xEE40f268487f9c2D664Aa66Cf5fD1B01d8b9fC3F', abiContract, wsProvider);
  // const contract = new ethers.Contract('0x8cbF1A9167F66B9B3310Aab56E4fEFc17514d23A', abiContract, wallet);
  // test events
  // Obtenir le bloc actuel
  // const currentBlock = await provider.getBlockNumber();
  // Chercher sur les 1000 derniers blocs par exemple
  // const fromBlock = Math.max(0, currentBlock - 1000);
  
  // const filter = contract.filters.CountIncremented();
  // Spécifier la plage de blocs
  // const events = await contract.queryFilter(filter, fromBlock, currentBlock);

  // const eventSignature = "CountIncremented(uint256)";
  // const eventHash = ethers.id(eventSignature);
  // console.log("Event signature:", eventSignature);
  // console.log("Event hash (keccak256):", eventHash);
  // console.log("Recherche d'événements du bloc", fromBlock, "au bloc", currentBlock);

  // for (const event of events) {
  //   console.log('Event:', event);
  // }

  contract.on('CountIncremented', (newCount, event) => {
    console.log('New event received!');
    console.log('even:', event);
    console.log('New count:', newCount.toString());
  });
}

async function createCronCallBackData() {
  console.log("wallet : ", wallet.address)
  try {
    console.log('createCronCallBackData en cours...');

    // let x = await wallet.sendTransaction({
    //   value: ethers.parseEther("500"),
    //   to: wallet2.address
    // });
    // await x.wait();

    const chronosAbi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/chronos/abi.json').toString()).abi;
    const contract = new ethers.Contract('0x0000000000000000000000000000000000000830', chronosAbi, wallet);
    const tx = await contract.createCallbackConditionedCron(
      "0x6E85bfd36946631d93e0D4a8d6eef2d88e4F4803",
      "callBack", // methodName
      0, // expirationBlock
      400000, // gasLimit
      ethers.parseUnits("2", "gwei"), // maxGasPrice
      ethers.parseEther("1")
    );
    console.log('Transaction envoyée, hash :', tx.hash);

    const receipt = await tx.wait();
    console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

    console.log(receipt);
  } catch (error) {
    console.error('Erreur lors de la createCron :', error);
  }
}

async function uploadLogo() {
  console.log("wallet : ", wallet.address)

  const logosAbi = JSON.parse(fs.readFileSync('../helios-chain/precompiles/logos/abi.json').toString()).abi;
  const contract = new ethers.Contract('0x0000000000000000000000000000000000000901', logosAbi, wallet);
  const tx = await contract.uploadLogo(
    "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");
  console.log('Transaction envoyée, hash :', tx.hash);

  contract.on('LogoUploaded', (hash, event) => {
    console.log('New event received!');
    console.log('even:', event);
    console.log('hash:', hash);
  });

  const receipt = await tx.wait();
  console.log('Transaction confirmée dans le bloc :', receipt.blockNumber);

  console.log(receipt, receipt.logs);
  // Parcours des logs pour trouver ton event
  const uploadedEvent = receipt.logs
    .map(log => {
      try {
        return contract.interface.parseLog(log);
      } catch (e) {
        return null;
      }
    })
    .filter(log => log && log.name === "LogoUploaded"); // nom exact de l'event dans le smart contract

  console.log("Event LogoUploaded :", uploadedEvent);
}

async function main() {
  // await createCronCallBackData();
  // await createCron();
  // await getEvents();
  // await getEventsCronCancelled();
  // await cancelCron();
  // await getEventsEVMCallScheduled();
  await create();
  //await fetch();
  //await delegate();
  //await addNewConsensusProposal();
  //await updateConsensusProposal();
  //await vote();
  // await undelegate();

  // await getEventsCronCreated();

  // await getRewards();

  // await sendToChain("5");
  // await sendToChain("5");
  // await setOrchestratorAddresses();
  // await addCounterpartyChainParams();
  // await uploadLogo();
}

main();