// SPDX-License-Identifier: MIT
// OpenZeppelin Contracts v4.3.2 (token/ERC20/presets/ERC20PresetMinterPauser.sol)
pragma solidity ^0.8.0;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Burnable.sol";
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Pausable.sol";
import "@openzeppelin/contracts/access/AccessControlEnumerable.sol";
import "@openzeppelin/contracts/utils/Context.sol";

/**
 * @dev {ERC20} token, including:
 *
 *  - ability for holders to burn (destroy) their tokens
 *  - a minter role that allows for token minting (creation)
 *  - a pauser role that allows to stop all token transfers
 *
 * This contract uses {AccessControl} to lock permissioned functions using the
 * different roles - head to its documentation for details.
 *
 * MODIFICATION: The deploying module can specify who gets the admin/minter/pauser roles
 * instead of automatically assigning them to msg.sender. This allows the user who called
 * the precompile to become the true owner of the token, following Solana/Ethereum standards.
 */
contract ERC20MinterBurnerDecimals is Context, AccessControlEnumerable, ERC20Burnable, ERC20Pausable {
    bytes32 public constant MINTER_ROLE = keccak256("MINTER_ROLE");
    bytes32 public constant PAUSER_ROLE = keccak256("PAUSER_ROLE");
    bytes32 public constant BURNER_ROLE = keccak256("BURNER_ROLE");
    
    uint8 private _decimals;

    /**
     * @dev Grants roles to specified addresses instead of msg.sender
     * @param name Token name
     * @param symbol Token symbol  
     * @param decimals_ Token decimals
     * @param initialOwner Who gets DEFAULT_ADMIN_ROLE (the user who called createErc20)
     * @param mintAuthority Who gets MINTER_ROLE (address(0) = no minter)
     * @param pauseAuthority Who gets PAUSER_ROLE (address(0) = no pauser)  
     * @param burnAuthority Who gets BURNER_ROLE (address(0) = no special burner)
     */
    constructor(
        string memory name, 
        string memory symbol, 
        uint8 decimals_,
        address initialOwner,      // The real owner (user who called createErc20)
        address mintAuthority,     // Who can mint new tokens
        address pauseAuthority,    // Who can pause/unpause transfers
        address burnAuthority      // Who can burn any tokens
    ) ERC20(name, symbol) {
        _decimals = decimals_;
        
        // Grant ownership to the user, not the module
        _setupRole(DEFAULT_ADMIN_ROLE, initialOwner);
        
        // Grant specialized roles only if specified (non-zero address)
        if (mintAuthority != address(0)) {
            _setupRole(MINTER_ROLE, mintAuthority);
        }
        if (pauseAuthority != address(0)) {
            _setupRole(PAUSER_ROLE, pauseAuthority);
        }  
        if (burnAuthority != address(0)) {
            _setupRole(BURNER_ROLE, burnAuthority);
        }
        
        // IMPORTANT: Module keeps temporary MINTER_ROLE for initial mint
        // This will be revoked after initial mint in the keeper
        _setupRole(MINTER_ROLE, _msgSender()); // _msgSender() = module temporarily
    }

    /**
     * @dev Sets `_decimals` as `decimals_` once at deployment
     */
    function _setupDecimals(uint8 decimals_) private {
        _decimals = decimals_;
    }

    /**
     * @dev Overrides the `decimals()` method with custom `_decimals`
     */
    function decimals() public view virtual override returns (uint8) {
        return _decimals;
    }

    /**
     * @dev Creates `amount` new tokens for `to`.
     *
     * See {ERC20-_mint}.
     *
     * Requirements:
     *
     * - the caller must have the `MINTER_ROLE`.
     */
    function mint(address to, uint256 amount) public virtual {
        require(hasRole(MINTER_ROLE, _msgSender()), "ERC20MinterBurnerDecimals: must have minter role to mint");
        _mint(to, amount);
    }

    /**
     * @dev Destroys `amount` tokens from `from`.
     *
     * See {ERC20-_burn}.
     *
     * Requirements:
     *
     * - the caller must have the `BURNER_ROLE`.
     */
    function burnCoins(address from, uint256 amount) public virtual {
        require(hasRole(BURNER_ROLE, _msgSender()), "ERC20MinterBurnerDecimals: must have burner role to burn");
        _burn(from, amount);
    }

    /**
     * @dev Pauses all token transfers.
     *
     * See {ERC20Pausable} and {Pausable-_pause}.
     *
     * Requirements:
     *
     * - the caller must have the `PAUSER_ROLE`.
     */
    function pause() public virtual {
        require(hasRole(PAUSER_ROLE, _msgSender()), "ERC20MinterBurnerDecimals: must have pauser role to pause");
        _pause();
    }

    /**
     * @dev Unpauses all token transfers.
     *
     * See {ERC20Pausable} and {Pausable-_unpause}.
     *
     * Requirements:
     *
     * - the caller must have the `PAUSER_ROLE`.
     */
    function unpause() public virtual {
        require(hasRole(PAUSER_ROLE, _msgSender()), "ERC20MinterBurnerDecimals: must have pauser role to unpause");
        _unpause();
    }

    function _beforeTokenTransfer(
        address from,
        address to,
        uint256 amount
    ) internal virtual override(ERC20, ERC20Pausable) {
        super._beforeTokenTransfer(from, to, amount);
    }
}