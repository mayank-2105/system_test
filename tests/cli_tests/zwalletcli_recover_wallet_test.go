package cli_tests

import (
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/stretchr/testify/require"

	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestRecoverWallet(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()

	t.Run("Recover wallet valid mnemonic", func(t *test.SystemTest) {
		validMnemonic := "pull floor crop best weasel suit solid gown" +
			" filter kitten loan absent noodle nation potato planet demise" +
			" online ten affair rich panel rent sell"

		output, err := recoverWalletFromMnemonic(t, configPath, validMnemonic, true)

		require.Nil(t, err, "error occurred recovering a wallet", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Wallet recovered!!", output[len(output)-1])
	})

	//FIXME: POSSIBLE BUG: Blank wallet created if mnemonic is invalid (same issue in missing mnemonic test)
	t.Run("Recover wallet invalid mnemonic", func(t *test.SystemTest) {
		inValidMnemonic := "floor crop best weasel suit solid gown" +
			" filter kitten loan absent noodle nation potato planet demise" +
			" online ten affair rich panel rent sell"

		output, err := recoverWalletFromMnemonic(t, configPath, inValidMnemonic, false)

		require.NotNil(t, err, "expected error to occur recovering a wallet", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		require.Equal(t, "Error: Invalid mnemonic", output[0])
	})

	t.Run("Recover wallet no mnemonic", func(t *test.SystemTest) {
		output, err := cliutils.RunCommandWithoutRetry("./zwallet recoverwallet --silent " +
			"--wallet " + escapedTestName(t) + "_wallet.json" + " " +
			"--configDir ./config --config " + configPath)

		require.NotNil(t, err, "expected error to occur recovering a wallet", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: Mnemonic not provided", output[0])
	})
}

func recoverWalletFromMnemonic(t *test.SystemTest, configPath, mnemonic string, retry bool) ([]string, error) {
	t.Logf("Recovering wallet from mnemonic...")
	cmd := "./zwallet recoverwallet " +
		"--silent --wallet " + escapedTestName(t) + "_wallet.json" + " " +
		"--configDir ./config --config " + configPath + " --mnemonic \"" + mnemonic + "\""

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
