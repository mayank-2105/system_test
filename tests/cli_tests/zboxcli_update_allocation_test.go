package cli_tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/stretchr/testify/require"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

var (
	createAllocationRegex = regexp.MustCompile(`^Allocation created: (.+)$`)
	updateAllocationRegex = regexp.MustCompile(`^Allocation updated with txId : [a-f0-9]{64}$`)
)

func TestUpdateAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()

	t.Run("Update Expiry Should Work", func(t *test.SystemTest) {
		allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
		expDuration := int64(1) // In hours

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"expiry":     fmt.Sprintf("%dh", expDuration),
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "Could not update "+
			"allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		allocations := parseListAllocations(t, configPath)
		ac, ok := allocations[allocationID]
		require.True(t, ok, "current allocation not found", allocationID, allocations)
		require.Equal(t, allocationBeforeUpdate.ExpirationDate+expDuration*3600, ac.ExpirationDate,
			fmt.Sprint("Expiration Time doesn't match: "+
				"Before:", allocationBeforeUpdate.ExpirationDate, "After:", ac.ExpirationDate),
		)
	})

	t.Run("Update Size Should Work", func(t *test.SystemTest) {
		allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
		size := int64(256)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       size,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "Could not update allocation "+
			"due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		allocations := parseListAllocations(t, configPath)
		ac, ok := allocations[allocationID]
		require.True(t, ok, "current allocation not found", allocationID, allocations)
		require.Equal(t, allocationBeforeUpdate.Size+size, ac.Size,
			fmt.Sprint("Size doesn't match: Before:", allocationBeforeUpdate.Size, "After:", ac.Size),
		)
	})

	t.Run("Update All Parameters Should Work", func(t *test.SystemTest) {
		allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
		expDuration := int64(1) // In hours
		size := int64(2048)

		params := createParams(map[string]interface{}{
			"allocation":   allocationID,
			"expiry":       fmt.Sprintf("%dh", expDuration),
			"size":         size,
			"update_terms": true,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		allocations := parseListAllocations(t, configPath)
		ac, ok := allocations[allocationID]
		require.True(t, ok, "current allocation not found", allocationID, allocations)
		require.Equal(t, allocationBeforeUpdate.ExpirationDate+expDuration*3600, ac.ExpirationDate)
		require.Equal(t, allocationBeforeUpdate.Size+size, ac.Size)
	})

	t.Run("Update Negative Expiry Should Not Work", func(t *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath)
		expDuration := int64(-30) // In minutes

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"expiry":     fmt.Sprintf("\"%dm\"", expDuration),
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.NotNil(t, err, "expected error while updating allocation expiry "+
			"by negative value", strings.Join(output, "\n"))
	})

	t.Run("Update Negative Size Should Work", func(t *test.SystemTest) {
		allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
		size := int64(-256)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       size,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		allocations := parseListAllocations(t, configPath)
		ac, ok := allocations[allocationID]
		require.True(t, ok, "current allocation not found", allocationID, allocations)
		require.Equal(t, allocationBeforeUpdate.Size+size, ac.Size,
			fmt.Sprint("Size doesn't match: Before:", allocationBeforeUpdate.Size, " After:", ac.Size),
		)
	})

	t.Run("Update All Negative Parameters Should Work", func(t *test.SystemTest) {
		allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
		size := int64(-512)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       size,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		allocations := parseListAllocations(t, configPath)
		ac, ok := allocations[allocationID]
		require.True(t, ok, "current allocation not found", allocationID, allocations)
		require.Equal(t, allocationBeforeUpdate.Size+size, ac.Size,
			fmt.Sprint("Size doesn't match: Before:", allocationBeforeUpdate.Size, " After:", ac.Size),
		)
	})

	t.Run("Update Size to less than occupied size should fail", func(t *test.SystemTest) {
		allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath) // alloc size is 10000

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, 2048) // uploading a file of size 2048
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/dir/",
			"localpath":  filename,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		size := int64(-9000) // reducing it by 9000 should fail since 2048 is being used
		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       size,
		})
		output, err = updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, output[0], "Error updating allocation:allocation_updating_failed: new allocation size is too small: 1000 < 1024")

		allocations := parseListAllocations(t, configPath)
		ac, ok := allocations[allocationID]
		require.True(t, ok, "current allocation not found", allocationID, allocations)
		require.Equal(t, allocationBeforeUpdate.Size, ac.Size,
			fmt.Sprint("Size doesn't match: Before:", allocationBeforeUpdate.Size, " After:", ac.Size),
		) // size should be unaffected
	})

	// FIXME expiry or size should be required params - should not bother sharders with an empty update
	t.Run("Update Nothing Should Fail", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
		})
		output, err := updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "Error updating allocation:allocation_updating_failed: update allocation changes nothing", output[0])
	})

	// TODO is it normal to create read pool?
	t.Run("Update Non-existent Allocation Should Fail", func(t *test.SystemTest) {
		allocationID := "123abc"

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"expiry":     "1h",
		})
		output, err := updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 3, "expected output length be at least 4", strings.Join(output, "\n"))
		require.Equal(t, "Error updating allocation:couldnt_find_allocation: Couldn't find the allocation required for update", output[3])
	})

	t.RunWithTimeout("Update Expired Allocation Should Fail", 60*time.Second, func(t *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath, map[string]interface{}{"expire": "2s"})

		time.Sleep(5 * time.Second)

		expDuration := int64(1) // In hours

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"expiry":     fmt.Sprintf("%dh", expDuration),
		})
		output, err := updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "Error updating allocation:allocation_updating_failed: can't update expired allocation", output[0])

		// Update the expired allocation's size
		size := int64(2048)

		params = createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       size,
		})
		output, err = updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "expected error updating allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "Error updating allocation:allocation_updating_failed: can't update expired allocation", output[0])
	})

	t.Run("Update Size To Less Than 1024 Should Fail", func(t *test.SystemTest) {
		allocationID, allocationBeforeUpdate := setupAndParseAllocation(t, configPath)
		size := -allocationBeforeUpdate.Size + 1023

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       fmt.Sprintf("\"%d\"", size),
		})
		output, err := updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "expected error updating "+
			"allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output "+
			"length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "Error updating allocation:allocation_updating_failed: new allocation size is too small: 1023 < 1024", output[0])
	})

	t.RunWithTimeout("Update Other's Allocation Should Fail", 60*time.Second, func(t *test.SystemTest) { // todo: too slow
		var otherAllocationID string

		myAllocationID := setupAllocation(t, configPath)

		// This test creates a separate wallet and allocates there, test nesting is required to create another wallet json file
		t.Run("Get Other Allocation ID", func(t *test.SystemTest) {
			otherAllocationID = setupAllocation(t, configPath)

			// Updating the otherAllocationID should work here
			size := int64(2048)

			params := createParams(map[string]interface{}{
				"allocation": otherAllocationID,
				"size":       size,
			})
			output, err := updateAllocation(t, configPath, params, true)

			require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
			require.Len(t, output, 1)
			assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		})

		// otherAllocationID should not be updatable from this level
		size := int64(2048)

		// First try updating with myAllocationID: should work
		params := createParams(map[string]interface{}{
			"allocation": myAllocationID,
			"size":       size,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "Could not update allocation due to error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// Then try updating with otherAllocationID: should not work
		params = createParams(map[string]interface{}{
			"allocation": otherAllocationID,
			"size":       size,
		})
		output, err = updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "expected error updating "+
			"allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error updating allocation:allocation_updating_failed: only owner can update the allocation", output[0])
	})

	t.Run("Update Mistake Expiry Parameter Should Fail", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)
		expiry := 1

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"expiry":     expiry,
		})
		output, err := updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "expected error updating "+
			"allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length "+
			"be at least 1", strings.Join(output, "\n"))
		expected := fmt.Sprintf(
			`Error: invalid argument "%v" for "--expiry" flag: time: missing unit in duration "%v"`,
			expiry, expiry,
		)
		require.Equal(t, expected, output[0])
	})

	t.Run("Update Mistake Size Parameter Should Fail", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)
		size := "ab"

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       size,
		})
		output, err := updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "expected error updating "+
			"allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at "+
			"least 1", strings.Join(output, "\n"))
		expected := fmt.Sprintf(
			`Error: invalid argument "%v" for "--size" flag: strconv.ParseInt: parsing "%v": invalid syntax`,
			size, size,
		)
		require.Equal(t, expected, output[0])
	})

	t.RunWithTimeout("Update Allocation flags for forbid and allow file_options should succeed", 2*time.Minute, func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath)

		// Forbid upload
		params := createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_upload": nil,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// get allocation
		alloc := getAllocation(t, allocationID)
		require.Equal(t, uint16(62), alloc.FileOptions) // 63 - 1 = 62 = 00111110

		// Forbid delete
		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_delete": nil,
		})
		output, err = updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(60), alloc.FileOptions) // 63 - 3 = 60 = 00011100

		// Forbid update
		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_update": nil,
		})
		output, err = updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(56), alloc.FileOptions) // 63 - 7 = 56 = 00111000

		// Forbid move
		params = createParams(map[string]interface{}{
			"allocation":  allocationID,
			"forbid_move": nil,
		})
		output, err = updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(48), alloc.FileOptions) // 63 - 15 = 48 = 00110000

		// Forbid copy
		params = createParams(map[string]interface{}{
			"allocation":  allocationID,
			"forbid_copy": nil,
		})
		output, err = updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(32), alloc.FileOptions) // 63 - 31 = 32 = 00100000

		// Forbid rename
		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_rename": nil,
		})
		output, err = updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(0), alloc.FileOptions) // 32 - 32 = 0 = 00000000

		// Allow upload
		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_upload": false,
		})

		output, err = updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(1), alloc.FileOptions) // 0 + 1 = 1 = 00000001

		// Allow delete
		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_delete": false,
		})
		output, err = updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(3), alloc.FileOptions) // 1 + 2 = 3 = 00000011

		// Allow update
		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_update": false,
		})
		output, err = updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(7), alloc.FileOptions) // 3 + 4 = 7 = 00000111

		// Allow move
		params = createParams(map[string]interface{}{
			"allocation":  allocationID,
			"forbid_move": false,
		})
		output, err = updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(15), alloc.FileOptions) // 7 + 8 = 15 = 00001111

		// Allow copy
		params = createParams(map[string]interface{}{
			"allocation":  allocationID,
			"forbid_copy": false,
		})
		output, err = updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(31), alloc.FileOptions) // 15 + 16 = 31 = 00011111

		// Allow rename
		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_rename": false,
		})
		output, err = updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// get allocation
		alloc = getAllocation(t, allocationID)
		require.Equal(t, uint16(63), alloc.FileOptions) // 31 + 32 = 63 = 00111111
	})

	t.Run("Updating same file options twice should fail", func(w *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath)

		// Forbid upload
		params := createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_upload": nil,
			"forbid_delete": nil,
			"forbid_move":   nil,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// Forbid upload
		params = createParams(map[string]interface{}{
			"allocation":    allocationID,
			"forbid_upload": nil,
			"forbid_delete": nil,
			"forbid_move":   nil,
		})
		output, err = updateAllocation(t, configPath, params, false)

		require.NotNil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Contains(t, output[0], "changes nothing")
	})

	t.Run("Update allocation set_third_party_extendable flag should work", func(t *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath)

		// set third party extendable
		params := createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// get allocation
		alloc := getAllocation(t, allocationID)
		require.True(t, alloc.ThirdPartyExtendable)
	})

	t.Run("Update allocation set_third_party_extendable flag should fail if third_party_extendable is already true", func(t *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath)

		// set third party extendable
		params := createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// get allocation
		alloc := getAllocation(t, allocationID)
		require.True(t, alloc.ThirdPartyExtendable)

		// set third party extendable
		params = createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
		})
		output, err = updateAllocation(t, configPath, params, true)

		require.NotNil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "changes nothing")
	})

	t.Run("Update allocation expand by third party if third_party_extendable = false should fail", func(t *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       1,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// get allocation
		alloc := getAllocation(t, allocationID)
		require.False(t, alloc.ThirdPartyExtendable)

		nonAllocOwnerWallet := escapedTestName(t) + "_NON_OWNER"

		output, err = registerWalletForName(t, configPath, nonAllocOwnerWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		// expand allocation
		params = createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       2,
		})
		output, err = updateAllocationWithWallet(t, nonAllocOwnerWallet, configPath, params, true)

		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "only owner can update the allocation")
	})

	t.Run("Update allocation expand by third party if third_party_extendable = true should succeed", func(t *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath)

		params := createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// get allocation
		alloc := getAllocation(t, allocationID)
		require.True(t, alloc.ThirdPartyExtendable)

		nonAllocOwnerWallet := escapedTestName(t) + "_NON_OWNER"

		output, err = registerWalletForName(t, configPath, nonAllocOwnerWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))
		_, err = executeFaucetWithTokensForWallet(t, nonAllocOwnerWallet, configPath, 3.0)
		require.Nil(t, err)

		// expand allocation
		params = createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       2,
			"expiry":     "24h",
		})
		output, err = updateAllocationWithWallet(t, nonAllocOwnerWallet, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))

		// get allocation
		allocUpdated := getAllocation(t, allocationID)
		require.Equal(t, int64(alloc.Size+2), allocUpdated.Size)

		expandedDuration, err := time.ParseDuration("24h")
		require.Nil(t, err)
		require.Equal(t, alloc.ExpirationDate+int64(expandedDuration.Seconds()), allocUpdated.ExpirationDate)
	})

	t.Run("Update allocation any other action than expand by third party regardless of third_party_extendable should fail", func(t *test.SystemTest) {
		allocationID, _ := setupAndParseAllocation(t, configPath)

		params := createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
		})
		output, err := updateAllocation(t, configPath, params, true)

		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])

		// get allocation
		alloc := getAllocation(t, allocationID)
		require.True(t, alloc.ThirdPartyExtendable)

		nonAllocOwnerWallet := escapedTestName(t) + "_NON_OWNER"

		output, err = registerWalletForName(t, configPath, nonAllocOwnerWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))
		_, err = executeFaucetWithTokensForWallet(t, nonAllocOwnerWallet, configPath, 3.0)
		require.Nil(t, err)

		// reduce allocation should fail
		params = createParams(map[string]interface{}{
			"allocation": allocationID,
			"size":       -100,
		})
		output, err = updateAllocationWithWallet(t, nonAllocOwnerWallet, configPath, params, false)
		require.NotNil(t, err, "no error updating allocation by third party", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "only owner can update the allocation")

		// set file_options or third_party_extendable should fail
		params = createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"forbid_upload":              nil,
			"forbid_update":              nil,
			"forbid_delete":              nil,
			"forbid_rename":              nil,
			"forbid_move":                nil,
			"forbid_copy":                nil,
			"set_third_party_extendable": nil,
		})
		output, err = updateAllocationWithWallet(t, nonAllocOwnerWallet, configPath, params, false)
		require.NotNil(t, err, "no error updating allocation by third party", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "only owner can update the allocation")

		// add blobber should fail
		params = createParams(map[string]interface{}{
			"allocation":     allocationID,
			"add_blobber":    "new_blobber_id",
			"remove_blobber": "blobber_id",
		})
		output, err = updateAllocationWithWallet(t, nonAllocOwnerWallet, configPath, params, false)
		require.NotNil(t, err, "no error updating allocation by third party", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "only owner can update the allocation")

		// set update_term should fail
		params = createParams(map[string]interface{}{
			"allocation":   allocationID,
			"update_terms": false,
		})
		output, err = updateAllocationWithWallet(t, nonAllocOwnerWallet, configPath, params, false)
		require.NotNil(t, err, "no error updating allocation by third party", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "only owner can update the allocation")

		// set lock should fail
		params = createParams(map[string]interface{}{
			"allocation": allocationID,
			"lock":       100,
		})
		output, err = updateAllocationWithWallet(t, nonAllocOwnerWallet, configPath, params, false)
		require.NotNil(t, err, "no error updating allocation by third party", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "only owner can update the allocation")

		// get allocation
		updatedAlloc := getAllocation(t, allocationID)
		require.Equal(t, alloc, updatedAlloc)
	})
}

func setupAndParseAllocation(t *test.SystemTest, cliConfigFilename string, extraParams ...map[string]interface{}) (string, climodel.Allocation) {
	allocationID := setupAllocation(t, cliConfigFilename, extraParams...)

	allocations := parseListAllocations(t, cliConfigFilename)
	allocation, ok := allocations[allocationID]
	require.True(t, ok, "current allocation not found", allocationID, allocations)

	return allocationID, allocation
}

func parseListAllocations(t *test.SystemTest, cliConfigFilename string) map[string]climodel.Allocation {
	output, err := listAllocations(t, cliConfigFilename)
	require.Nil(t, err, "list allocations failed", err, strings.Join(output, "\n"))
	require.Len(t, output, 1)

	var allocations []*climodel.Allocation
	err = json.NewDecoder(strings.NewReader(output[0])).Decode(&allocations)
	require.Nil(t, err, "error deserializing JSON", err)

	allocationMap := make(map[string]climodel.Allocation)

	for _, ac := range allocations {
		allocationMap[ac.ID] = *ac
	}

	return allocationMap
}

func setupAllocation(t *test.SystemTest, cliConfigFilename string, extraParams ...map[string]interface{}) string {
	return setupAllocationWithWallet(t, escapedTestName(t), cliConfigFilename, extraParams...)
}

func setupAllocationWithWallet(t *test.SystemTest, walletName, cliConfigFilename string, extraParams ...map[string]interface{}) string {
	faucetTokens := 1.0
	// Then create new allocation
	options := map[string]interface{}{"expire": "1h", "size": "10000", "lock": "0.5"}

	// Add additional parameters if available
	// Overwrite with new parameters when available
	for _, params := range extraParams {
		// Extract parameters unrelated to upload
		if tokenStr, ok := params["tokens"]; ok {
			token, err := strconv.ParseFloat(fmt.Sprintf("%v", tokenStr), 64)
			require.Nil(t, err)
			faucetTokens = token
			delete(params, "tokens")
		}
		for k, v := range params {
			options[k] = v
		}
	}
	// First create a wallet and run faucet command
	output, err := registerWalletForName(t, cliConfigFilename, walletName)
	require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

	output, err = executeFaucetWithTokensForWallet(t, walletName, cliConfigFilename, faucetTokens)
	require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

	output, err = createNewAllocationForWallet(t, walletName, cliConfigFilename, createParams(options))
	require.Nil(t, err, "create new allocation failed", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	// Get the allocation ID and return it
	allocationID, err := getAllocationID(output[0])
	require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

	return allocationID
}

func assertOutputMatchesAllocationRegex(t *test.SystemTest, re *regexp.Regexp, str string) {
	match := re.FindStringSubmatch(str)
	require.True(t, len(match) > 0, "expected allocation to match regex", re, str)
}

func getAllocationID(str string) (string, error) {
	match := createAllocationRegex.FindStringSubmatch(str)
	if len(match) < 2 {
		return "", errors.New("allocation match not found")
	}
	return match[1], nil
}

func getAllocationCost(str string) (float64, error) {
	allocationCostInOutput, err := strconv.ParseFloat(strings.Fields(str)[5], 64)
	if err != nil {
		return 0.0, err
	}

	unit := strings.Fields(str)[6]
	allocationCostInZCN := unitToZCN(allocationCostInOutput, unit)

	return allocationCostInZCN, nil
}

func createParams(params map[string]interface{}) string {
	var builder strings.Builder

	for k, v := range params {
		if v == nil {
			_, _ = builder.WriteString(fmt.Sprintf("--%s ", k))
		} else if reflect.TypeOf(v).String() == "bool" {
			_, _ = builder.WriteString(fmt.Sprintf("--%s=%v ", k, v))
		} else {
			_, _ = builder.WriteString(fmt.Sprintf("--%s %v ", k, v))
		}
	}
	return strings.TrimSpace(builder.String())
}

func createKeyValueParams(params map[string]string) string {
	keys := "--keys \""
	values := "--values \""
	first := true
	for k, v := range params {
		if first {
			first = false
		} else {
			keys += ","
			values += ","
		}
		keys += " " + k
		values += " " + v
	}
	keys += "\""
	values += "\""
	return keys + " " + values
}

func updateAllocation(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return updateAllocationWithWallet(t, escapedTestName(t), cliConfigFilename, params, retry)
}

func updateAllocationWithWallet(t *test.SystemTest, wallet, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Logf("Updating allocation...")
	cmd := fmt.Sprintf(
		"./zbox updateallocation %s --silent --wallet %s "+
			"--configDir ./config --config %s --lock 0.2",
		params,
		wallet+"_wallet.json",
		cliConfigFilename,
	)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func listAllocations(t *test.SystemTest, cliConfigFilename string) ([]string, error) {
	cliutils.Wait(t, 5*time.Second)
	t.Logf("Listing allocations...")
	cmd := fmt.Sprintf(
		"./zbox listallocations --json --silent "+
			"--wallet %s --configDir ./config --config %s",
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)
	return cliutils.RunCommand(t, cmd, 3, time.Second*2)
}

// executeFaucetWithTokens executes faucet command with given tokens.
// Tokens greater than or equal to 10 are considered to be 1 token by the system.
func executeFaucetWithTokens(t *test.SystemTest, cliConfigFilename string, tokens float64) ([]string, error) {
	return executeFaucetWithTokensForWallet(t, escapedTestName(t), cliConfigFilename, tokens)
}

// executeFaucetWithTokensForWallet executes faucet command with given tokens and wallet.
// Tokens greater than or equal to 10 are considered to be 1 token by the system.
func executeFaucetWithTokensForWallet(t *test.SystemTest, wallet, cliConfigFilename string, tokens float64) ([]string, error) {
	t.Logf("Executing faucet...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zwallet faucet --methodName "+
		"pour --tokens %f --input {} --silent --wallet %s_wallet.json --configDir ./config --config %s",
		tokens,
		wallet,
		cliConfigFilename,
	), 3, time.Second*5)
}
