package types

// Event types and attribute keys for the feedistribution module
const (
	// Event types
	EventTypeRegisterContract = "register_contract"
	EventTypeDistributeFees   = "distribute_fees"
	EventTypeRegisterRevenue  = "register_revenue"
	EventTypeUpdateWithdrawer = "update_withdrawer"
	EventTypeDeleteRevenue    = "delete_revenue"

	// Attribute keys
	AttributeKeyContract          = "contract"
	AttributeKeyDeployer          = "deployer"
	AttributeKeyAmount            = "amount"
	AttributeKeyWithdrawer        = "withdrawer"
	AttributeKeyDistributionType  = "distribution_type"
	AttributeKeyWithdrawerAddress = "withdrawer_address"
)
