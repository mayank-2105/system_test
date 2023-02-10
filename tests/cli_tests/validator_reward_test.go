package cli_tests

import (
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	"testing"
)

func TestValidatorReward(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()

	t.RunSequentially("Test Validator Reward", func(t *test.SystemTest) {
		walletID := "f91edef60ce5c6ca4763432a67d3991d99bb50ecd16638f56f424e3d0a6bf5e0"

		wallet, err := getBalanceForWallet(t, configPath, walletID)
		fmt.Println("wallet", wallet)
		if err != nil {
			return
		}
		//allocSize := int64(1 * MB)
		//fileSize := int64(512 * KB)
		//
		//allocationID := setupAllocation(t, configPath, map[string]interface{}{
		//	"size":   allocSize,
		//	"parity": 1,
		//	"data":   1,
		//})
		//
		//filename := generateRandomTestFileName(t)
		//err := createFileWithSize(filename, fileSize)
		//require.Nil(t, err)
		//
		//output, err := uploadFile(t, configPath, map[string]interface{}{
		//	"allocation": allocationID,
		//	"remotepath": "/",
		//	"localpath":  filename,
		//}, true)
		//require.Nil(t, err, strings.Join(output, "\n"))
		//require.Len(t, output, 2)
		//
		//expected := fmt.Sprintf(
		//	"Status completed callback. Type = application/octet-stream. Name = %s",
		//	filepath.Base(filename),
		//)
		//require.Equal(t, expected, output[1])
	})
}
