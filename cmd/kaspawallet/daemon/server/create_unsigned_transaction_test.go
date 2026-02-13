package server

import (
	"context"
	"testing"
	"time"

	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/daemon/pb"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/keys"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet/serialization"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/txscript"
	"github.com/kaspanet/kaspad/domain/consensus/utils/utxo"
	"github.com/kaspanet/kaspad/domain/dagconfig"
	"github.com/kaspanet/kaspad/util/txmass"
)

type stubWalletRPCClient struct {
	virtualDAAScore uint64
}

func (s *stubWalletRPCClient) GetMempoolEntry(string, bool, bool) (*appmessage.GetMempoolEntryResponseMessage, error) {
	return nil, nil
}

func (s *stubWalletRPCClient) GetFeeEstimate() (*appmessage.GetFeeEstimateResponseMessage, error) {
	return nil, nil
}

func (s *stubWalletRPCClient) GetBlockDAGInfo() (*appmessage.GetBlockDAGInfoResponseMessage, error) {
	return &appmessage.GetBlockDAGInfoResponseMessage{
		VirtualDAAScore: s.virtualDAAScore,
	}, nil
}

func (s *stubWalletRPCClient) GetUTXOsByAddresses([]string) (*appmessage.GetUTXOsByAddressesResponseMessage, error) {
	return nil, nil
}

func (s *stubWalletRPCClient) GetBalancesByAddresses([]string) (*appmessage.GetBalancesByAddressesResponseMessage, error) {
	return nil, nil
}

func (s *stubWalletRPCClient) GetMempoolEntriesByAddresses([]string, bool, bool) (*appmessage.GetMempoolEntriesByAddressesResponseMessage, error) {
	return nil, nil
}

func (s *stubWalletRPCClient) SubmitTransaction(*appmessage.RPCTransaction, string, bool) (*appmessage.SubmitTransactionResponseMessage, error) {
	return nil, nil
}

func (s *stubWalletRPCClient) SubmitTransactionReplacement(*appmessage.RPCTransaction, string) (*appmessage.SubmitTransactionReplacementResponseMessage, error) {
	return nil, nil
}

func TestCreateUnsignedTransactionsMultiPayment(t *testing.T) {
	params := &dagconfig.SimnetParams
	mnemonic, err := libkaspawallet.CreateMnemonic()
	if err != nil {
		t.Fatalf("CreateMnemonic: %+v", err)
	}
	xpub, err := libkaspawallet.MasterPublicKeyFromMnemonic(params, mnemonic, false)
	if err != nil {
		t.Fatalf("MasterPublicKeyFromMnemonic: %+v", err)
	}

	keysFile := &keys.File{
		ExtendedPublicKeys: []string{xpub},
		MinimumSignatures:  1,
		CosignerIndex:      0,
		ECDSA:              false,
	}
	rpcClient := &stubWalletRPCClient{virtualDAAScore: 10_000}
	s := &server{
		rpcClient:           rpcClient,
		backgroundRPCClient: rpcClient,
		params:              params,
		coinbaseMaturity:    1000,
		keysFile:            keysFile,
		shutdown:            make(chan struct{}),
		forceSyncChan:       make(chan struct{}, 1),
		addressSet:          make(walletAddressSet),
		txMassCalculator: txmass.NewCalculator(
			params.MassPerTxByte,
			params.MassPerScriptPubKeyByte,
			params.MassPerSigOp),
		usedOutpoints:      map[externalapi.DomainOutpoint]time.Time{},
		nextSyncStartIndex: 1,
	}
	s.firstSyncDone.Store(true)

	sourceWalletAddress := &walletAddress{
		index:         1,
		cosignerIndex: 0,
		keyChain:      libkaspawallet.ExternalKeychain,
	}
	sourceAddress, err := libkaspawallet.Address(params, keysFile.ExtendedPublicKeys, keysFile.MinimumSignatures, s.walletAddressPath(sourceWalletAddress), keysFile.ECDSA)
	if err != nil {
		t.Fatalf("Address(source): %+v", err)
	}
	sourceScriptPublicKey, err := txscript.PayToAddrScript(sourceAddress)
	if err != nil {
		t.Fatalf("PayToAddrScript(source): %+v", err)
	}
	transactionID := *externalapi.NewDomainTransactionIDFromByteArray(&[externalapi.DomainHashSize]byte{0x01})
	s.utxosSortedByAmount = []*walletUTXO{{
		Outpoint: &externalapi.DomainOutpoint{
			TransactionID: transactionID,
			Index:         0,
		},
		UTXOEntry: utxo.NewUTXOEntry(100_000_000, sourceScriptPublicKey, false, 0),
		address:   sourceWalletAddress,
	}}

	toAddress1, err := libkaspawallet.Address(params, keysFile.ExtendedPublicKeys, keysFile.MinimumSignatures, "m/0/2", keysFile.ECDSA)
	if err != nil {
		t.Fatalf("Address(to1): %+v", err)
	}
	toAddress2, err := libkaspawallet.Address(params, keysFile.ExtendedPublicKeys, keysFile.MinimumSignatures, "m/0/3", keysFile.ECDSA)
	if err != nil {
		t.Fatalf("Address(to2): %+v", err)
	}
	amount1 := uint64(5_000_000)
	amount2 := uint64(7_000_000)
	response, err := s.CreateUnsignedTransactions(context.Background(), &pb.CreateUnsignedTransactionsRequest{
		ToAddresses: []string{toAddress1.String(), toAddress2.String()},
		Amounts:     []uint64{amount1, amount2},
		FeePolicy: &pb.FeePolicy{
			FeePolicy: &pb.FeePolicy_ExactFeeRate{ExactFeeRate: 1},
		},
		UseExistingChangeAddress: true,
	})
	if err != nil {
		t.Fatalf("CreateUnsignedTransactions(multi): %+v", err)
	}

	if len(response.UnsignedTransactions) != 1 {
		t.Fatalf("expected 1 unsigned transaction, got %d", len(response.UnsignedTransactions))
	}

	unsignedTransaction, err := serialization.DeserializePartiallySignedTransaction(response.UnsignedTransactions[0])
	if err != nil {
		t.Fatalf("DeserializePartiallySignedTransaction: %+v", err)
	}

	if len(unsignedTransaction.Tx.Outputs) != 3 {
		t.Fatalf("expected 3 outputs (2 payments + change), got %d", len(unsignedTransaction.Tx.Outputs))
	}
	if unsignedTransaction.Tx.Outputs[0].Value != amount1 {
		t.Fatalf("expected first payment output value %d, got %d", amount1, unsignedTransaction.Tx.Outputs[0].Value)
	}
	if unsignedTransaction.Tx.Outputs[1].Value != amount2 {
		t.Fatalf("expected second payment output value %d, got %d", amount2, unsignedTransaction.Tx.Outputs[1].Value)
	}
}
