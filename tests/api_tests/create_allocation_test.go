package api_tests

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/go-resty/resty/v2" //nolint

	"github.com/0chain/system_test/internal/api/model"
	"github.com/stretchr/testify/require"
)

func TestCreateAllocation(t *testing.T) {
	t.Parallel()

	t.Run("Create allocation API call should be successful given a valid request", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)
		response, confirmation := executeFaucet(t, registeredWallet, keyPair)
		require.Nil(t, response)
		require.Equal(t, api_client.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		blobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 147483648, 2, 2, time.Minute*20)
		blobberRequirements.Blobbers = blobbers
		transactionResponse, confirmation := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		require.Equal(t, api_client.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		allocation := getAllocation(t, transactionResponse.Entity.Hash)

		require.NotNil(t, allocation)
	})
}

func createAllocation(t *testing.T, wallet *model.Wallet, keyPair *model.KeyPair, blobberRequirements model.BlobberRequirements) (*model.TransactionResponse, *model.Confirmation) {
	t.Logf("Creating allocation...")
	txnDataString, err := json.Marshal(model.SmartContractTxnData{Name: "new_allocation_request", InputArgs: blobberRequirements})
	require.Nil(t, err)
	allocationRequest := model.Transaction{
		PublicKey:        keyPair.PublicKey.SerializeToHexStr(),
		TxnOutputHash:    "",
		TransactionValue: 1000000000,
		TransactionType:  1000,
		TransactionFee:   0,
		TransactionData:  string(txnDataString),
		ToClientId:       api_client.StorageSmartContractAddress,
		CreationDate:     time.Now().Unix(),
		ClientId:         wallet.ClientID,
		Version:          "1.0",
		TransactionNonce: wallet.Nonce + 1,
	}

	allocationTransaction := executeTransaction(t, &allocationRequest, keyPair)
	confirmation, _ := confirmTransaction(t, wallet, allocationTransaction.Entity, 2*time.Minute)

	return allocationTransaction, confirmation
}

func getAllocation(t *testing.T, allocationId string) *model.Allocation {
	allocation, httpResponse, err := getAllocationWithoutAssertion(t, allocationId)

	require.NotNil(t, allocation, "Allocation was unexpectedly nil! with http response [%s]", httpResponse)
	require.Nil(t, err, "Unexpected error [%s] occurred getting balance with http response [%s]", err, httpResponse)
	require.Equal(t, api_client.HttpOkStatus, httpResponse.Status())

	return allocation
}

func getAllocationWithoutAssertion(t *testing.T, allocationId string) (*model.Allocation, *resty.Response, error) { //nolint
	t.Logf("Retrieving allocation...")
	balance, httpResponse, err := api_client.v1ScrestAllocation(t, allocationId, nil)
	return balance, httpResponse, err
}
