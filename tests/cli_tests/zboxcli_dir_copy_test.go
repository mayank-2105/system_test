package cli_tests

import (
	"encoding/json"
	"strings"
	"testing"

	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

func TestCopyDir(t *testing.T) {
	t.Parallel()

	t.Run("copy nested dir", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)
		defer createAllocationTestTeardown(t, allocationID)

		dirname := "/rootdir"
		output, err := createDir(t, configPath, allocationID, dirname, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname+" directory created", output[0])

		dirname2 := "/rootdir/nested_directory"
		output, err = createDir(t, configPath, allocationID, dirname2, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname2+" directory created", output[0])

		dirname3 := "/destination_direcory"
		output, err = createDir(t, configPath, allocationID, dirname3, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname3+" directory created", output[0])

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var directories []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &directories)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		require.Len(t, directories, 3, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		require.Contains(t, directories, climodel.AllocationFile{Name: "rootdir", Path: "/rootdir", Type: "d"}, "rootdir expected to be created")
		require.Contains(t, directories, climodel.AllocationFile{Name: "nested_directory", Path: "/rootdir/nested_directory", Type: "d"}, "nested_directory expected to be created")
		require.Contains(t, directories, climodel.AllocationFile{Name: "destination_direcory", Path: "/destination_direcory", Type: "d"}, "destination_direcory expected to be created")

		output, err = copyFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": dirname2,
			"destpath":   "/destination_direcory/nested_directory",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		// FIXME! copy directory is broken. Test must be fixed when copy directory fauture is fixed
		t.Log("FIXME! copy directory is broken. Test must be fixed when copy directory fauture is fixed")
		require.Equal(t, "Copy failed: Commit consensus failed", output[0])
	})

	t.Run("copy root dir", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)
		defer createAllocationTestTeardown(t, allocationID)

		dirname := "/rootdir"
		output, err := createDir(t, configPath, allocationID, dirname, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname+" directory created", output[0])

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var directories []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &directories)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		require.Len(t, directories, 1, "Expecting rootdir created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		require.Contains(t, directories, climodel.AllocationFile{Name: "rootdir", Path: "/rootdir", Type: "d"}, "rootdir expected to be created")

		output, err = copyFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": dirname,
			"destpath":   "/rootdir_copy",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		// FIXME! copy directory is broken. Test must be fixed when copy directory fauture is fixed
		t.Log("FIXME! copy directory is broken. Test must be fixed when copy directory fauture is fixed")
		require.Equal(t, "Copy failed: Commit consensus failed", output[0])
	})
}
