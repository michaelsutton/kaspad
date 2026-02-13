package main

import (
	"context"
	"fmt"
	"os"

	"github.com/kaspanet/kaspad/cmd/kaspawallet/daemon/client"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/daemon/pb"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/daemon/server"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/utils"
)

func createUnsignedTransaction(conf *createUnsignedTransactionConfig) error {
	daemonClient, tearDown, err := client.Connect(conf.DaemonAddress)
	if err != nil {
		return err
	}
	defer tearDown()

	ctx, cancel := context.WithTimeout(context.Background(), daemonTimeout)
	defer cancel()

	sendAmountsSompi := make([]uint64, len(conf.SendAmount))
	if !conf.IsSendAll {
		for i, sendAmount := range conf.SendAmount {
			sendAmountsSompi[i], err = utils.KasToSompi(sendAmount)
			if err != nil {
				return err
			}
		}
	}

	var feePolicy *pb.FeePolicy
	if conf.FeeRate > 0 {
		feePolicy = &pb.FeePolicy{
			FeePolicy: &pb.FeePolicy_ExactFeeRate{
				ExactFeeRate: conf.FeeRate,
			},
		}
	} else if conf.MaxFeeRate > 0 {
		feePolicy = &pb.FeePolicy{
			FeePolicy: &pb.FeePolicy_MaxFeeRate{MaxFeeRate: conf.MaxFeeRate},
		}
	} else if conf.MaxFee > 0 {
		feePolicy = &pb.FeePolicy{
			FeePolicy: &pb.FeePolicy_MaxFee{MaxFee: conf.MaxFee},
		}
	}

	request := &pb.CreateUnsignedTransactionsRequest{
		From:                     conf.FromAddresses,
		ToAddresses:              conf.ToAddress,
		Amounts:                  sendAmountsSompi,
		IsSendAll:                conf.IsSendAll,
		UseExistingChangeAddress: conf.UseExistingChangeAddress,
		FeePolicy:                feePolicy,
	}
	// Keep legacy fields populated for compatibility with older daemons.
	if len(conf.ToAddress) > 0 {
		request.Address = conf.ToAddress[0]
	}
	if len(sendAmountsSompi) > 0 {
		request.Amount = sendAmountsSompi[0]
	}

	response, err := daemonClient.CreateUnsignedTransactions(ctx, request)
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "Created unsigned transaction")
	fmt.Println(server.EncodeTransactionsToHex(response.UnsignedTransactions))

	return nil
}
