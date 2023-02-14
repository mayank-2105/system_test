package cli_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidatorReward(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

	t.RunSequentiallyWithTimeout("Test Validator Reward", (50*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 1,
		})

		// get validator wallet balance
		validatorDelegateWalletBalance, _ := getBalanceForWallet(t, configPath, validatorWallet)

		startBlock := getLatestFinalizedBlock(t)

		remotepath := "/dir/"
		filesize := 2 * MB
		filename := generateRandomTestFileName(t)

		err = createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		//sleep for 1 minute to allow the challenges to be created
		//time.Sleep(1 * time.Minute)

		//sleep for 1 minute to allow the challenges to be created
		time.Sleep(60 * time.Second)

		//fetch the latest block in the chain
		endBlock := getLatestFinalizedBlock(t)

		blocks := getBlockContainingBlobberReward(t, startBlock, endBlock)

		// block transactions
		for _, block := range blocks {
			for _, tx := range block.Block.Transactions {
				fmt.Println("transaction_data : ", tx.TransactionData)
			}
		}

		// list validators
		vldtr, _ := listValidators(t, configPath, "--json")
		// bind validators with Validator
		var validatorList []Validator
		err = json.Unmarshal([]byte(strings.Join(vldtr, "")), &validatorList)
		fmt.Println("validators", vldtr)

		// collect reward for each validator
		for _, validator := range validatorList {
			output, err = collectRewards(t, configPath, createParams(map[string]interface{}{
				"provider_type": "validator",
				"provider_id":   validator.ValidatorId,
			}), true)

			fmt.Println("collect rewards for validator", output)
			fmt.Println(err)
		}

		validatorDelegateWalletBalanceAfter, _ := getBalanceForWallet(t, configPath, validatorWallet)

		// check if the validator wallet balance is greater than before
		require.Greater(t, validatorDelegateWalletBalanceAfter, validatorDelegateWalletBalance, "validator wallet balance should be greater than before")
	})
}

type Validator struct {
	ValidatorId              string  `json:"validator_id"`
	Url                      string  `json:"url"`
	DelegateWallet           string  `json:"delegate_wallet"`
	MinStake                 int64   `json:"min_stake"`
	MaxStake                 int64   `json:"max_stake"`
	NumDelegates             int     `json:"num_delegates"`
	ServiceCharge            float64 `json:"service_charge"`
	StakeTotal               int     `json:"stake_total"`
	UnstakeTotal             int     `json:"unstake_total"`
	TotalServiceCharge       int     `json:"total_service_charge"`
	UncollectedServiceCharge int     `json:"uncollected_service_charge"`
}
