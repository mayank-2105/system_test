package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"
)

func TestCreateAllocation(t *testing.T) {
	t.Parallel()

	t.Run("Create allocation API call should be successful given a valid request", func(t *testing.T) {
		t.Parallel()

		wallet := apiClient.RegisterWallet(t)
		apiClient.ExecuteFaucet(t, wallet, client.TxSuccessfulStatus)

		allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, nil, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)
		apiClient.GetAllocation(t, allocationID, client.HttpOkStatus)
	})
}
