package cli_tests

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/stretchr/testify/require"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestRegisterWallet(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()

	t.Run("Register wallet outputs expected", func(t *test.SystemTest) {
		output, err := registerWallet(t, configPath)

		require.Nil(t, err, "An error occurred registering a wallet", strings.Join(output, "\n"))
		require.Len(t, output, 4)
		require.Equal(t, "ZCN wallet created", output[0])
		require.Equal(t, "Creating related read pool for storage smart-contract...", output[1])
		require.Equal(t, "Read pool created successfully", output[2])
		require.Equal(t, "Wallet registered", output[3])
	})

	t.Run("Get wallet outputs expected", func(t *test.SystemTest) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "An error occurred registering a wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)

		require.Nil(t, err, "An error occurred retrieving a wallet", strings.Join(output, "\n"))
		require.NotNil(t, wallet.ClientID)
		require.NotNil(t, wallet.ClientPublicKey)
		require.NotNil(t, wallet.EncryptionPublicKey)
	})

	t.Run("Balance call fails due to zero ZCN in wallet", func(t *test.SystemTest) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "An error occurred registering a wallet", strings.Join(output, "\n"))

		output, err = getBalance(t, configPath)
		ensureZeroBalance(t, output, err)
	})

	t.Run("Balance of 1 is returned after faucet execution", func(t *test.SystemTest) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "An error occurred registering a wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		matcher := regexp.MustCompile("Execute faucet smart contract success with txn : {2}([a-f0-9]{64})$")
		require.Regexp(t, matcher, output[0], "Faucet execution output did not match expected")

		txnID := matcher.FindAllStringSubmatch(output[0], 1)[0][1]
		output, err = verifyTransaction(t, configPath, txnID)
		require.Nil(t, err, "Could not verify faucet transaction", strings.Join(output, "\n"))

		require.Len(t, output, 3)
		require.Equal(t, "Transaction verification success", output[0])
		require.Equal(t, "TransactionStatus: 1", output[1])
		require.Greater(t, len(output[2]), 0, output[2])

		output, err = getBalance(t, configPath)
		require.Nil(t, err, "An error occurred retrieving wallet balance", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 1.000 ZCN \([0-9.]+ USD\)$`), output[0])
	})
}

func registerWalletAndLockReadTokens(t *test.SystemTest, cliConfigFilename string) error {
	_, err := registerWalletForName(t, cliConfigFilename, escapedTestName(t))
	if err != nil {
		return err
	}
	var tokens float64 = 2
	_, err = executeFaucetWithTokens(t, cliConfigFilename, tokens)
	if err != nil {
		return err
	}

	// Lock half the tokens for read pool
	readPoolParams := createParams(map[string]interface{}{
		"tokens": tokens / 2,
	})
	_, err = readPoolLock(t, cliConfigFilename, readPoolParams, true)

	return err
}

func registerWallet(t *test.SystemTest, cliConfigFilename string) ([]string, error) {
	return registerWalletForName(t, cliConfigFilename, escapedTestName(t))
}

func registerWalletForName(t *test.SystemTest, cliConfigFilename, name string) ([]string, error) {
	t.Logf("Registering wallet...")
	return cliutils.RunCommand(t, "./zbox register --silent "+
		"--wallet "+name+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func registerWalletForNameAndLockReadTokens(t *test.SystemTest, cliConfigFilename, name string) {
	var tokens = 2.0
	registerWalletWithTokens(t, cliConfigFilename, name, tokens)
	readPoolParams := createParams(map[string]interface{}{
		"tokens": tokens / 2,
	})
	_, err := readPoolLockWithWallet(t, name, cliConfigFilename, readPoolParams, true)
	require.NoErrorf(t, err, "error occurred when locking read pool for %s", name)
}

func registerWalletWithTokens(t *test.SystemTest, cliConfigFilename, name string, tokens float64) {
	_, err := registerWalletForName(t, cliConfigFilename, name)
	require.NoErrorf(t, err, "register wallet %s", name)
	_, err = executeFaucetWithTokensForWallet(t, name, cliConfigFilename, tokens)
	require.NoErrorf(t, err, "get tokens for wallet %s", name)
}

func getBalance(t *test.SystemTest, cliConfigFilename string) ([]string, error) {
	cliutils.Wait(t, 5*time.Second)
	return getBalanceForWallet(t, cliConfigFilename, escapedTestName(t))
}

func ensureZeroBalance(t *test.SystemTest, output []string, err error) {
	if err != nil {
		require.Len(t, output, 1)
		require.Equal(t, "Failed to get balance:", output[0])
		return
	}
	require.Equal(t, "Balance: 0 SAS (0.00 USD)", output[0])
}

func getBalanceForWallet(t *test.SystemTest, cliConfigFilename, wallet string) ([]string, error) {
	return cliutils.RunCommand(t, "./zwallet getbalance --silent "+
		"--wallet "+wallet+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func getWallet(t *test.SystemTest, cliConfigFilename string) (*climodel.Wallet, error) {
	return getWalletForName(t, cliConfigFilename, escapedTestName(t))
}

func getWalletForName(t *test.SystemTest, cliConfigFilename, name string) (*climodel.Wallet, error) {
	t.Logf("Getting wallet...")
	output, err := cliutils.RunCommand(t, "./zbox getwallet --json --silent "+
		"--wallet "+name+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)

	if err != nil {
		return nil, err
	}

	require.Len(t, output, 1)

	var wallet *climodel.Wallet

	err = json.Unmarshal([]byte(output[0]), &wallet)
	if err != nil {
		t.Errorf("failed to unmarshal the result into wallet")
		return nil, err
	}

	return wallet, err
}

func verifyTransaction(t *test.SystemTest, cliConfigFilename, txn string) ([]string, error) {
	t.Logf("Verifying transaction...")
	return cliutils.RunCommand(t, "./zwallet verify --silent --wallet "+escapedTestName(t)+""+
		"_wallet.json"+" --hash "+txn+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func escapedTestName(t *test.SystemTest) string {
	replacer := strings.NewReplacer("/", "-", "\"", "-", ":", "-", "(", "-",
		")", "-", "<", "LESS_THAN", ">", "GREATER_THAN", "|", "-", "*", "-",
		"?", "-")
	return replacer.Replace(t.Name())
}
