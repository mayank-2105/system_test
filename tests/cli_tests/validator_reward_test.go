package cli_tests

import (
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
	fmt.Println(output)
	require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

	t.RunSequentiallyWithTimeout("Test Validator Reward", (5*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 1,
		})

		remotepath := "/dir/"
		filesize := 2 * MB
		filename := generateRandomTestFileName(t)

		lfb := getLatestFinalizedBlock(t)
		fmt.Println("lfb", lfb.NumTxns)

		err = createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			// fetch the latest block in the chain
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")
		fmt.Println("wallet", wallet)

		//sleep for 1 minute to allow the challenges to be created
		time.Sleep(1 * time.Minute)

		//fetch the latest block in the chain
		lfb = getLatestFinalizedBlock(t)
		fmt.Println("lfb", lfb.NumTxns)
		//
		//traverse through each block to see if there is any reward transaction
		//
		//sleep for 1 minute to allow the challenges to be created
		//time.Sleep(1 * time.Minute)

		// list all validators and print their balances
		output, err = listValidators(t, configPath, "--json")
		fmt.Println("validators", output)
		//
		//validatorWallet := getValidatorWallet(t, output[0])
		//fmt.Println("validator wallet", validatorWallet)
		//

		//f, _ := getBalanceForWallet(t, configPath, validatorWallet)
		//fmt.Println("balance", f)
	})
}
