package api_tests

import (
	"crypto/rand"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

func TestReplaceBlobber(t *testing.T) {
	t.Parallel()

	t.Run("Replace blobber in allocation, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.HttpOkStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		oldBlobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		apiClient.UpdateAllocationBlobbers(t, wallet, newBlobberID, oldBlobberID, allocationID, client.TxSuccessfulStatus)

		allocation = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersAfter := len(allocation.Blobbers)

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
		require.True(t, isBlobberExist(newBlobberID, allocation.Blobbers))
	})

	t.Run("Replace blobber with the same one in allocation, shouldn't work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.HttpOkStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		oldBlobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		apiClient.UpdateAllocationBlobbers(t, wallet, oldBlobberID, oldBlobberID, allocationID, client.TxUnsuccessfulStatus)

		allocation = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersAfter := len(allocation.Blobbers)

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})

	t.Run("Replace blobber with incorrect blobber ID of an old blobber, shouldn't work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.HttpOkStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "Old blobber ID contains zero value")

		result, err := rand.Int(rand.Reader, big.NewInt(10))
		require.Nil(t, err)

		apiClient.UpdateAllocationBlobbers(t, wallet, newBlobberID, result.String(), allocationID, client.TxUnsuccessfulStatus)

		allocation = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersAfter := len(allocation.Blobbers)

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
	})

	t.Run("Check token accounting of a blobber replacing in allocation, should work", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t, "", "", nil, true, client.HttpOkStatus)
		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.HttpOkStatus)

		allocation := apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersBefore := len(allocation.Blobbers)

		oldBlobberID := getFirstUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, oldBlobberID, "Old blobber ID contains zero value")

		newBlobberID := getNotUsedStorageNodeID(allocationBlobbers.Blobbers, allocation.Blobbers)
		require.NotZero(t, newBlobberID, "New blobber ID contains zero value")

		walletBalance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		balanceBeforeAllocationUpdate := walletBalance.Balance

		apiClient.UpdateAllocationBlobbers(t, wallet, newBlobberID, oldBlobberID, allocationID, client.TxUnsuccessfulStatus)

		walletBalance = apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
		balanceAfterAllocationUpdate := walletBalance.Balance

		allocation = apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
		numberOfBlobbersAfter := len(allocation.Blobbers)

		require.Equal(t, numberOfBlobbersAfter, numberOfBlobbersBefore)
		require.Greater(t, balanceBeforeAllocationUpdate, balanceAfterAllocationUpdate)
	})
}
