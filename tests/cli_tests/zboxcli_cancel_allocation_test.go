package cli_tests

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/stretchr/testify/require"

	cliutils "github.com/0chain/system_test/internal/cli/util"
)

var (
	cancelAllocationRegex = regexp.MustCompile(`^Allocation canceled with txId : [a-f0-9]{64}$`)
)

func TestCancelAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()

	t.Run("Cancel allocation immediately should work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)

		output, err := cancelAllocation(t, configPath, allocationID, true)
		require.NoError(t, err, "cancel allocation failed but should succeed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, cancelAllocationRegex, output[0])
	})

	t.Run("No allocation param should fail", func(t *test.SystemTest) {
		cmd := fmt.Sprintf(
			"./zbox alloc-cancel --silent "+
				"--wallet %s --configDir ./config --config %s",
			escapedTestName(t)+"_wallet.json",
			configPath,
		)

		output, err := cliutils.RunCommandWithoutRetry(cmd)
		require.Error(t, err, "expected error canceling allocation", strings.Join(output, "\n"))
		require.Len(t, output, 4)
		require.Equal(t, "Error: allocation flag is missing", output[len(output)-1])
	})

	t.Run("Cancel Other's Allocation Should Fail", func(t *test.SystemTest) {
		otherAllocationID := setupAllocationWithWallet(t, escapedTestName(t)+"_other_wallet.json", configPath)

		// otherAllocationID should not be cancelable from this level
		output, err := cancelAllocation(t, configPath, otherAllocationID, false)

		require.Error(t, err, "expected error canceling allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "Error creating allocation:alloc_cancel_failed: only owner can cancel an allocation", output[len(output)-1])
	})

	t.Run("Cancel Non-existent Allocation Should Fail", func(t *test.SystemTest) {
		allocationID := "123abc"

		output, err := cancelAllocation(t, configPath, allocationID, false)

		require.Error(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 3, "expected output length be at least 4", strings.Join(output, "\n"))
		//FIXME: error is incorrect, should be error canceling allocation see https://github.com/0chain/zboxcli/issues/240
		require.Equal(t, "Error creating allocation:alloc_cancel_failed: value not present", output[len(output)-1])
	})

	t.Run("Cancel Expired Allocation Should Fail", func(t *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath, map[string]interface{}{
			"expire": "2s",
		})

		time.Sleep(5 * time.Second)
		allocations := parseListAllocations(t, configPath)
		ac, ok := allocations[allocationID]
		require.True(t, ok, "current allocation not found", allocationID, allocations)
		require.LessOrEqual(t, ac.ExpirationDate, time.Now().Unix())

		// Cancel the expired allocation
		output, err := cancelAllocation(t, configPath, allocationID, false)
		require.Error(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))

		require.Equal(t, "Error creating allocation:alloc_cancel_failed: trying to cancel expired allocation", output[0])
	})
}

func cancelAllocation(t *test.SystemTest, cliConfigFilename, allocationID string, retry bool) ([]string, error) {
	t.Logf("Canceling allocation...")
	cmd := fmt.Sprintf(
		"./zbox alloc-cancel --allocation %s --silent "+
			"--wallet %s --configDir ./config --config %s",
		allocationID,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
