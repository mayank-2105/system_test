package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutil "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestSharderFeeRewards(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)

	if !confirmDebugBuild(t) {
		t.Skip("miner block rewards test skipped as it requires a debug event database")
	}

	// Take a snapshot of the chains miners, then wait a few seconds, take another snapshot.
	// Examine the rewards paid between the two snapshot and confirm the self-consistency
	// of the block reward payments
	//
	// Each round a random miner is chosen to receive the block reward.
	// The miner's service charge is used to determine the fraction received by the miner's wallet.
	//
	// The remaining block reward is then distributed amongst the miner's delegates.
	//
	// A subset of the delegates chosen at random to receive a portion of the block reward.
	// The total received by each stake pool is proportional to the tokens they have locked
	// wither respect to the total locked by the chosen delegate pools.
	t.RunWithTimeout("Miner share of block fees and rewards", 240*time.Second, func(t *test.SystemTest) {
		walletId := initialiseTest(t, escapedTestName(t)+"_TARGET", true)
		output, err := executeFaucetWithTokens(t, configPath, 10)
		require.NoError(t, err, "faucet execution failed", strings.Join(output, "\n"))
		output, err = executeFaucetWithTokens(t, configPath, 10)
		require.NoError(t, err, "faucet execution failed", strings.Join(output, "\n"))

		sharderUrl := getSharderUrl(t)
		sharderIds := getSortedSharderIds(t, sharderUrl)
		require.True(t, len(sharderIds) > 1, "this test needs at least two sharders")

		// todo piers remove
		//tokens := []float64{1, 0.5}
		//_ = createSharderStakePools(t, sharderIds, tokens)
		//t.Cleanup(func() {
		//	cleanupFunc()
		//})

		beforeSharders := getNodes(t, sharderIds, sharderUrl)

		// ------------------------------------
		//cliutils.Wait(t, 2*time.Second)
		const numPaidTransactions = 1
		const fee = 0.1
		for i := 0; i < numPaidTransactions; i++ {
			output, err := sendTokens(t, configPath, walletId, 0.5, escapedTestName(t), fee)
			require.Nil(t, err, "error sending tokens", strings.Join(output, "\n"))
		}
		// ------------------------------------

		afterSharders := getNodes(t, sharderIds, sharderUrl)

		// we add rewards at the end of the round, and they don't appear until the next round
		startRound := beforeSharders.Nodes[0].RoundServiceChargeLastUpdated + 1
		endRound := afterSharders.Nodes[0].RoundServiceChargeLastUpdated + 1
		for i := range beforeSharders.Nodes {
			if startRound < beforeSharders.Nodes[i].RoundServiceChargeLastUpdated {
				startRound = beforeSharders.Nodes[i].RoundServiceChargeLastUpdated
			}
			if endRound > afterSharders.Nodes[i].RoundServiceChargeLastUpdated {
				endRound = afterSharders.Nodes[i].RoundServiceChargeLastUpdated
			}
			t.Logf("miner %s delegates pools %d", beforeSharders.Nodes[i].ID, len(beforeSharders.Nodes[i].Pools))
		}
		t.Logf("start round %d, end round %d", startRound, endRound)

		history := cliutil.NewHistory(startRound, endRound)
		history.Read(t, sharderUrl, true)

		minerScConfig := getMinerScMap(t)
		numSharderDelegatesRewarded := int(minerScConfig["num_sharder_delegates_rewarded"])
		var numShardersRewarded int
		if len(sharderIds) > int(minerScConfig["num_sharders_rewarded"]) {
			numShardersRewarded = int(minerScConfig["num_sharders_rewarded"])
		} else {
			numShardersRewarded = len(sharderIds)
		}
		minerShare := minerScConfig["share_ratio"]
		// Each round one miner is chosen to receive a block reward.
		// The winning miner is stored in the block object.
		// The reward payments retrieved from the provider reward table.
		// The amount of the reward is a fraction of the block reward allocated to miners each
		// round. The fraction is the miner's service charge. If the miner has
		// no stake pools then the reward becomes the full block reward.
		//
		// Firstly we confirm the self-consistency of the block and reward tables.
		// We calculate the change in the miner rewards during and confirm that this
		// equals the total of the reward payments read from the provider rewards table.
		for i, id := range sharderIds {
			var blockRewards, feeRewards int64
			for round := beforeSharders.Nodes[i].RoundServiceChargeLastUpdated + 1; round <= afterSharders.Nodes[i].RoundServiceChargeLastUpdated; round++ {
				feesPerSharder := int64(float64(history.FeesForRound(t, round)) / float64(numShardersRewarded))
				roundHistory := history.RoundHistory(t, round)
				for _, pReward := range roundHistory.ProviderRewards {
					if pReward.ProviderId != id {
						continue
					}
					switch pReward.RewardType {
					case climodel.FeeRewardSharder:
						require.Greater(testSetup, feesPerSharder, int64(0), "fee reward with no fees")
						var fees int64
						if len(beforeSharders.Nodes[i].StakePool.Pools) > 0 {
							fees = int64(float64(feesPerSharder) * beforeSharders.Nodes[i].Settings.ServiceCharge * (1 - minerShare))
						} else {
							fees = int64(float64(feesPerSharder) * (1 - minerShare))
						}
						if fees != pReward.Amount {
							fmt.Println("fees", fees, "reward", pReward.Amount)
						}
						require.InDeltaf(t, fees, pReward.Amount, delta,
							"incorrect service charge %v for round %d"+
								" service charge should be fees %d multiplied by service ratio %v."+
								"length stake pools %d",
							pReward.Amount, round, fees, beforeSharders.Nodes[i].Settings.ServiceCharge,
							len(beforeSharders.Nodes[i].StakePool.Pools))
						feeRewards += pReward.Amount
					case climodel.BlockRewardSharder:
						blockRewards += pReward.Amount
					default:
						require.Failf(t, "reward type %s is not available for miners", pReward.RewardType.String())
					}
				}
			}
			actualReward := afterSharders.Nodes[i].Reward - beforeSharders.Nodes[i].Reward
			if actualReward != blockRewards+feeRewards {
				fmt.Println("piers actual rewards", actualReward, "block rewards", blockRewards, "fee rewards", feeRewards)
			}

			require.InDeltaf(t, actualReward, blockRewards+feeRewards, delta,
				"rewards expected %v, change in sharder reward during the test is %v", actualReward, blockRewards+feeRewards)
		}
		t.Log("finished testing sharders")

		// Each round there is a fee, there should be exactly num_sharders_rewarded sharder fee reward payment.
		for round := startRound + 1; round <= endRound-1; round++ {
			if history.FeesForRound(t, round) == 0 {
				continue
			}
			roundHistory := history.RoundHistory(t, round)
			shardersPaid := make(map[string]bool)
			for _, pReward := range roundHistory.ProviderRewards {
				if pReward.RewardType == climodel.FeeRewardSharder {
					_, found := shardersPaid[pReward.ProviderId]
					require.Falsef(t, found, "sharder %s receives more than one block reward on round %d", pReward.ProviderId, round)
					shardersPaid[pReward.ProviderId] = true
				}
			}
			require.Equal(t, numShardersRewarded, len(shardersPaid),
				"mismatch between expected count of sharders rewarded and actual number on round %d", round)

		}
		t.Log("about to test delegate pools")

		// Each round there is a fee each sharder rewarded should have num_sharder_delegates_rewarded of
		// their delegates rewarded, or all delegates if less.
		for round := history.From(); round <= history.To(); round++ {
			if history.FeesForRound(t, round) == 0 {
				continue
			}
			roundHistory := history.RoundHistory(t, round)
			for i, id := range sharderIds {
				poolsPaid := make(map[string]bool)
				for poolId := range beforeSharders.Nodes[i].Pools {
					for _, dReward := range roundHistory.DelegateRewards {
						if dReward.RewardType != climodel.FeeRewardSharder || dReward.PoolID != poolId {
							continue
						}
						_, found := poolsPaid[poolId]
						if found {
							require.Falsef(t, found, "pool %s should have only received block reward once, round %d", poolId, round)
						}
						poolsPaid[poolId] = true
					}
				}
				numShouldPay := numSharderDelegatesRewarded
				if numShouldPay > len(beforeSharders.Nodes[i].Pools) {
					numShouldPay = len(beforeSharders.Nodes[i].Pools)
				}
				require.Len(t, poolsPaid, numShouldPay,
					"should pay %d pools for shader %s on round %d; %d pools actually paid",
					numShouldPay, id, round, len(poolsPaid))
			}
		}

		// Each round confirm payments to delegates or the blocks winning miner.
		// There should be exactly `num_miner_delegates_rewarded` delegates rewarded each round,
		// or all delegates if less.
		//
		// Delegates should be rewarded in proportional to their locked tokens.
		// We check the self-consistency of the reward payments each round using
		// the delegate reward table.
		//
		// Next we compare the actual change in rewards to each miner delegate, with the
		// change expected from the delegate reward table.

		for i, id := range sharderIds {
			numPools := len(afterSharders.Nodes[i].StakePool.Pools)
			rewards := make(map[string]int64, numPools)
			for poolId := range afterSharders.Nodes[i].StakePool.Pools {
				rewards[poolId] = 0
			}
			for round := beforeSharders.Nodes[i].RoundServiceChargeLastUpdated + 1; round <= afterSharders.Nodes[i].RoundServiceChargeLastUpdated; round++ {
				fees := history.FeesForRound(t, round)
				poolsBlockRewarded := make(map[string]int64)
				roundHistory := history.RoundHistory(t, round)
				for _, dReward := range roundHistory.DelegateRewards {
					if dReward.ProviderID != id {
						continue
					}
					_, isSharderPool := rewards[dReward.PoolID]
					require.Truef(testSetup, isSharderPool, "round %d, invalid pool id, reward %v", round, dReward)
					switch dReward.RewardType {
					case climodel.FeeRewardSharder:
						require.Greater(testSetup, fees, int64(0), "fee reward with no fees")
						_, found := poolsBlockRewarded[dReward.PoolID]
						require.False(t, found, "delegate pool %s paid a fee reward more than once on round %d",
							dReward.PoolID, round)
						poolsBlockRewarded[dReward.PoolID] = dReward.Amount
						rewards[dReward.PoolID] += dReward.Amount
					case climodel.BlockRewardSharder:
						rewards[dReward.PoolID] += dReward.Amount
					default:
						require.Failf(t, "mismatched reward type",
							"reward type %s not paid to miner delegate pools", dReward.RewardType)
					}
				}
				if fees > 0 {
					confirmPoolPayments(
						t,
						delegateSharderFeesRewards(
							numShardersRewarded,
							fees,
							beforeSharders.Nodes[i].Settings.ServiceCharge,
							1-minerShare,
						),
						poolsBlockRewarded,
						afterSharders.Nodes[i].StakePool.Pools,
						numSharderDelegatesRewarded,
					)
				}
			}
			for poolId := range afterSharders.Nodes[i].StakePool.Pools {
				actualReward := afterSharders.Nodes[i].StakePool.Pools[poolId].Reward - beforeSharders.Nodes[i].StakePool.Pools[poolId].Reward
				require.InDeltaf(t, actualReward, rewards[poolId], delta,
					"poolID %s, rewards expected %v change in pools reward during test", poolId, rewards[poolId],
				)
			}
		}
	})
}

func delegateSharderFeesRewards(numberSharders int, fee int64, serviceCharge, sharderShare float64) int64 {
	fmt.Println("num sharders", numberSharders, "fee", fee,
		"service charge", serviceCharge, "share", sharderShare, "result",
		int64(float64(fee)*(1-serviceCharge)*sharderShare/float64(numberSharders)))
	return int64(float64(fee) * (1 - serviceCharge) * sharderShare / float64(numberSharders))
}