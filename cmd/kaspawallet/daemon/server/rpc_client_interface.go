package server

import (
	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/infrastructure/network/rpcclient"
)

type walletRPCClient interface {
	GetMempoolEntry(txID string, includeOrphanPool bool, filterTransactionPool bool) (*appmessage.GetMempoolEntryResponseMessage, error)
	GetFeeEstimate() (*appmessage.GetFeeEstimateResponseMessage, error)
	GetBlockDAGInfo() (*appmessage.GetBlockDAGInfoResponseMessage, error)
	GetUTXOsByAddresses(addresses []string) (*appmessage.GetUTXOsByAddressesResponseMessage, error)
	GetBalancesByAddresses(addresses []string) (*appmessage.GetBalancesByAddressesResponseMessage, error)
	GetMempoolEntriesByAddresses(addresses []string, includeOrphanPool bool, filterTransactionPool bool) (*appmessage.GetMempoolEntriesByAddressesResponseMessage, error)
	SubmitTransaction(transaction *appmessage.RPCTransaction, transactionID string, allowOrphan bool) (*appmessage.SubmitTransactionResponseMessage, error)
	SubmitTransactionReplacement(transaction *appmessage.RPCTransaction, transactionID string) (*appmessage.SubmitTransactionReplacementResponseMessage, error)
}

var _ walletRPCClient = (*rpcclient.RPCClient)(nil)
