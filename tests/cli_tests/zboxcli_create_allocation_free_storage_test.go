package cli_tests

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/0chain/system_test/internal/api/util/crypto"

	"github.com/herumi/bls-go-binary/bls"

	"github.com/stretchr/testify/require"

	climodel "github.com/0chain/system_test/internal/cli/model"
)

const (
	chainID                     = "0afc093ffb509f059c55478bc1a60351cef7b4e9c008a53a6cc8241ca8617dfe"
	storageSmartContractAddress = `6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7`
	minerSmartContractAddress   = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9"
	txnTypeSmartContract        = 1000 // A smart contract transaction type

	freeTokensIndividualLimit = 10.0
	freeTokensTotalLimit      = 100.0
	configKeyReadPoolFraction = "free_allocation_settings.read_pool_fraction"
)

func TestCreateAllocationFreeStorage(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	err := bls.Init(bls.CurveFp254BNb)
	require.NoError(t, err, "Error initializing BLS")

	if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
		t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
	}

	assigner := escapedTestName(t) + "_ASSIGNER"

	// register SC owner wallet
	output, err := registerWalletForName(t, configPath, scOwnerWallet)
	require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

	// register assigner wallet
	output, err = registerWalletForName(t, configPath, assigner)
	require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

	// Open the wallet file themselves to get private key for signing data
	ownerWallet := readWalletFile(t, "./config/"+scOwnerWallet+"_wallet.json")
	assignerWallet := readWalletFile(t, "./config/"+assigner+"_wallet.json")

	// necessary cli call to generate wallet to avoid polluting logs of succeeding cli calls
	output, err = registerWallet(t, configPath)
	require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

	output, err = getStorageSCConfig(t, configPath, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Greater(t, len(output), 0, strings.Join(output, "\n"))

	cfg, _ := keyValuePairStringToMap(output)

	// miners list
	output, err = getMiners(t, configPath)
	require.Nil(t, err, "get miners failed", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	var miners climodel.NodeList
	err = json.Unmarshal([]byte(output[0]), &miners)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[0], err)
	require.NotEmpty(t, miners.Nodes, "No miners found: %v", strings.Join(output, "\n"))

	freeAllocAssignerTxn := freeAllocationAssignerTxn(t, ownerWallet, assignerWallet)
	err = sendTxn(miners, freeAllocAssignerTxn)
	require.Nil(t, err, "Error sending txn to miners: %v", output[0], err)

	output, err = verifyTransaction(t, configPath, freeAllocAssignerTxn.Hash)
	require.Nil(t, err, "Could not verify commit transaction", strings.Join(output, "\n"))
	require.Len(t, output, 3)
	require.Equal(t, "Transaction verification success", output[0])
	require.Equal(t, "TransactionStatus: 1", output[1])
	require.Greater(t, len(output[2]), 0, output[2])

	t.RunSequentiallyWithTimeout("Create free storage from marker with accounting", 60*time.Second, func(t *test.SystemTest) {
		recipient := escapedTestName(t)

		// register recipient wallet
		output, err = registerWalletForName(t, configPath, recipient)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		recipientWallet, err := getWalletForName(t, configPath, recipient)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		marker := climodel.FreeStorageMarker{
			Recipient:  recipientWallet.ClientID,
			FreeTokens: 5,
			Timestamp:  time.Now().Unix(),
		}

		forSignatureBytes, err := json.Marshal(&marker)
		require.Nil(t, err, "Could not marshal marker")

		data := hex.EncodeToString(forSignatureBytes)
		rawHash, err := hex.DecodeString(data)
		require.Nil(t, err, "failed to decode hex %s", data)
		require.NotNil(t, rawHash, "failed to decode hex %s", data)
		secretKey := crypto.ToSecretKey(t, assignerWallet)
		marker.Signature = crypto.Sign(t, string(rawHash), secretKey)

		marker.Assigner = assignerWallet.ClientID

		forFileBytes, err := json.Marshal(marker)
		require.Nil(t, err, "Could not marshal marker")

		markerFile := "./config/" + recipient + "_MARKER.json"

		err = os.WriteFile(markerFile, forFileBytes, 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationForWallet(t, recipient, configPath, createParams(map[string]interface{}{"free_storage": markerFile}))
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		matcher := regexp.MustCompile("Allocation created: ([a-f0-9]{64})")
		require.Regexp(t, matcher, output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		readPoolFraction, err := strconv.ParseFloat(cfg[configKeyReadPoolFraction], 64)
		require.Nil(t, err, "Read pool fraction config is not float: %s", cfg[configKeyReadPoolFraction])

		wantReadPoolFraction := marker.FreeTokens * readPoolFraction
		wantWritePoolToken := marker.FreeTokens - wantReadPoolFraction

		allocation := getAllocation(t, allocationID)
		require.Equal(t, ConvertToValue(wantWritePoolToken), allocation.WritePool, "Expected write pool amount not met", strings.Join(output, "\n"))
		require.Equal(t, uint16(63), allocation.FileOptions, "Expected file_options to be 63 (all allowed) by default", strings.Join(output, "\n"))

		readPool := getReadPoolInfo(t)
		require.Equal(t, ConvertToValue(wantReadPoolFraction), readPool.Balance, "Read Pool balance must be equal to locked amount")
	})

	t.RunSequentially("Create free storage with malformed marker should fail", func(t *test.SystemTest) {
		// register recipient wallet
		output, err = registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		markerFile := "./config/" + escapedTestName(t) + "_MARKER.json"

		err = os.WriteFile(markerFile, []byte("bad marker json"), 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(map[string]interface{}{"free_storage": markerFile}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "unmarshalling markerinvalid character 'b' looking for beginning of value", output[0])
	})

	t.RunSequentially("Create free storage with invalid marker contents should fail", func(t *test.SystemTest) {
		// register recipient wallet
		output, err = registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		markerFile := "./config/" + escapedTestName(t) + "_MARKER.json"

		err = os.WriteFile(markerFile, []byte(`{"invalid_marker":true}`), 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(map[string]interface{}{"free_storage": markerFile}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "Error creating free allocation: free_allocation_failed: marker can be used only by its recipient", output[0])
	})

	t.RunSequentially("Create free storage with invalid marker signature should fail", func(t *test.SystemTest) {
		recipient := escapedTestName(t)

		// register recipient wallet
		output, err = registerWalletForName(t, configPath, recipient)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		recipientWallet, err := getWalletForName(t, configPath, recipient)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		marker := climodel.FreeStorageMarker{
			Recipient:  recipientWallet.ClientID,
			FreeTokens: 5,
			Timestamp:  time.Now().Unix(),
		}

		marker.Signature = "badsignature"
		marker.Assigner = assignerWallet.ClientID

		forFileBytes, err := json.Marshal(marker)
		require.Nil(t, err, "Could not marshal marker")

		markerFile := "./config/" + recipient + "_MARKER.json"

		err = os.WriteFile(markerFile, forFileBytes, 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(map[string]interface{}{"free_storage": markerFile}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1)
		require.Equal(t, "Error creating free allocation: free_allocation_failed: marker verification failed: encoding/hex: invalid byte: U+0073 's'", output[0])
	})

	t.RunSequentially("Create free storage with wrong recipient wallet should fail", func(t *test.SystemTest) {
		recipientCorrect := escapedTestName(t) + "_RECIPIENT"

		// register correct recipient wallet
		output, err = registerWalletForName(t, configPath, recipientCorrect)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		recipientWallet, err := getWalletForName(t, configPath, recipientCorrect)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		// register this wallet
		output, err = registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		marker := climodel.FreeStorageMarker{
			Recipient:  recipientWallet.ClientID,
			FreeTokens: 5,
			Timestamp:  time.Now().Unix(),
		}

		forSignatureBytes, err := json.Marshal(&marker)
		require.Nil(t, err, "Could not marshal marker")

		data := hex.EncodeToString(forSignatureBytes)
		rawHash, err := hex.DecodeString(data)
		require.Nil(t, err, "failed to decode hex %s", data)
		require.NotNil(t, rawHash, "failed to decode hex %s", data)
		secretKey := crypto.ToSecretKey(t, assignerWallet)
		marker.Signature = crypto.Sign(t, string(rawHash), secretKey)
		marker.Assigner = assignerWallet.ClientID

		forFileBytes, err := json.Marshal(marker)
		require.Nil(t, err, "Could not marshal marker")

		markerFile := "./config/" + recipientCorrect + "_MARKER.json"

		err = os.WriteFile(markerFile, forFileBytes, 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(map[string]interface{}{"free_storage": markerFile}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Equal(t, 1, len(output), strings.Join(output, "\n"))
		require.Equal(t, "Error creating free allocation: free_allocation_failed: marker can be used only by its recipient", output[0])
	})

	t.RunSequentially("Create free storage with tokens exceeding assigner's individual limit should fail", func(t *test.SystemTest) {
		recipient := escapedTestName(t)

		// register recipient wallet
		output, err = registerWalletForName(t, configPath, recipient)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		recipientWallet, err := getWalletForName(t, configPath, recipient)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		marker := climodel.FreeStorageMarker{
			Recipient:  recipientWallet.ClientID,
			FreeTokens: freeTokensIndividualLimit + 1,
			Timestamp:  time.Now().Unix(),
		}

		forSignatureBytes, err := json.Marshal(&marker)
		require.Nil(t, err, "Could not marshal marker")

		data := hex.EncodeToString(forSignatureBytes)
		rawHash, err := hex.DecodeString(data)
		require.Nil(t, err, "failed to decode hex %s", data)
		require.NotNil(t, rawHash, "failed to decode hex %s", data)
		secretKey := crypto.ToSecretKey(t, assignerWallet)
		marker.Signature = crypto.Sign(t, string(rawHash), secretKey)
		marker.Assigner = assignerWallet.ClientID

		forFileBytes, err := json.Marshal(marker)
		require.Nil(t, err, "Could not marshal marker")

		markerFile := "./config/" + recipient + "_MARKER.json"

		err = os.WriteFile(markerFile, forFileBytes, 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(map[string]interface{}{"free_storage": markerFile}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1)
		require.Equal(t, "Error creating free allocation: free_allocation_failed: marker verification failed: 110000000000 exceeded permitted free storage  100000000000", output[0])
	})
}

func readWalletFile(t *test.SystemTest, file string) *climodel.WalletFile {
	wallet := &climodel.WalletFile{}

	f, err := os.Open(file)
	require.Nil(t, err, "wallet file %s not found", file)

	ownerWalletBytes, err := io.ReadAll(f)
	require.Nil(t, err, "error reading wallet file %s", file)

	err = json.Unmarshal(ownerWalletBytes, wallet)
	require.Nil(t, err, "error marshaling wallet content")

	return wallet
}

func sendTxn(miners climodel.NodeList, txn *climodel.Transaction) error {
	var (
		err error
		wg  sync.WaitGroup
	)

	for i := range miners.Nodes {
		wg.Add(1)
		go func(node climodel.Node) {
			defer wg.Done()
			_, apiErr := apiPutTransaction(getNodeBaseURL(node.Host, node.Port), txn)
			if apiErr != nil {
				err = apiErr
			}
		}(miners.Nodes[i])
	}
	wg.Wait()

	return err
}

func apiPutTransaction(minerBaseURL string, txn *climodel.Transaction) (*http.Response, error) {
	txnData, err := json.Marshal(txn)
	if err != nil {
		return nil, err
	}

	return http.Post(minerBaseURL+"/v1/transaction/put", "application/json", bytes.NewBuffer(txnData))
}

func freeAllocationAssignerTxn(t *test.SystemTest, from, assigner *climodel.WalletFile) *climodel.Transaction {
	txn := &climodel.Transaction{}
	txn.Version = "1.0"
	txn.ClientId = from.ClientID
	txn.CreationDate = time.Now().Unix()
	txn.ChainId = chainID
	txn.PublicKey = from.ClientKey
	txn.TransactionType = txnTypeSmartContract
	txn.ToClientId = storageSmartContractAddress
	txn.TransactionValue = 0
	ret, err := getNonceForWallet(t, configPath, scOwnerWallet, true)
	require.Nil(t, err, "error fetching minerNodeDelegate nonce")
	nonceStr := strings.Split(ret[0], ":")[1]
	nonce, err := strconv.ParseInt(strings.Trim(nonceStr, " "), 10, 64)
	require.Nil(t, err, "error converting nonce to in")
	txn.TransactionNonce = int(nonce) + 1

	input := map[string]interface{}{
		"name":             assigner.ClientID,
		"public_key":       assigner.ClientKey,
		"individual_limit": freeTokensIndividualLimit,
		"total_limit":      freeTokensTotalLimit,
	}

	sn := climodel.TransactionData{Name: "add_free_storage_assigner", Input: input}
	snBytes, err := json.Marshal(sn)
	require.Nil(t, err, "error marshaling smart contract data")

	txn.TransactionData = string(snBytes)

	txn.Hash = crypto.Sha3256([]byte(fmt.Sprintf("%d:%d:%s:%s:%d:%s",
		txn.CreationDate,
		txn.TransactionNonce,
		txn.ClientId,
		txn.ToClientId,
		txn.TransactionValue,
		crypto.Sha3256([]byte(txn.TransactionData)))))

	hashToSign, err := hex.DecodeString(txn.Hash)
	if err != nil {
		log.Fatalln(err)
	}

	keypair := crypto.GenerateKeys(t, from.Mnemonic)

	txn.Signature = keypair.PrivateKey.Sign(string(hashToSign)).
		SerializeToHexStr()

	return txn
}
