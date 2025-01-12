package cli_tests

import (
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutil "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestSharderBlockRewards(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)
	t.Skip("Skip till chain-side bugs are resolved")

	// Take a snapshot of the chains sharders, then wait a few seconds, take another snapshot.
	// Examine the rewards paid between the two snapshot and confirm the self-consistency
	// of the block reward payments
	//
	// Each round we choose num_sharders_rewarded random sharders to receive the block reward.
	// The sharder's service charge is used to determine the fraction received by the sharder's wallet.
	//
	// The remaining block reward is then distributed amongst num_sharder_delegates_rewarded of the sharder's delegates.
	//
	// A subset of the delegates chosen at random to receive a portion of the block reward.
	// The total received by each stake pool is proportional to the tokens they have locked
	// wither respect to the total locked by the chosen delegate pools.
	t.RunWithTimeout("Sharder share of block rewards", 500*time.Second, func(t *test.SystemTest) {
		_ = initialiseTest(t, escapedTestName(t)+"_TARGET", true)
		if !confirmDebugBuild(t) {
			t.Skip("sharder block rewards test skipped as it requires a debug event database")
		}

		sharderUrl := getSharderUrl(t)
		sharderIds := getSortedSharderIds(t, sharderUrl)
		require.True(t, len(sharderIds) > 1, "this test needs at least two sharders")

		beforeSharders := getNodes(t, sharderIds, sharderUrl)

		// ----------------------------------- w
		time.Sleep(time.Second * 3)
		// ----------------------------------=

		afterSharders := getNodes(t, sharderIds, sharderUrl)

		// we add rewards at the end of the round, and they don't appear until the next round
		startRound, endRound := getStartAndEndRounds(
			t, nil, nil, beforeSharders.Nodes, afterSharders.Nodes,
		)

		time.Sleep(time.Second) // give time for last round to be saved

		history := cliutil.NewHistory(startRound, endRound)
		history.Read(t, sharderUrl, false)

		balanceSharderRewards(
			t, startRound, endRound, sharderIds, beforeSharders.Nodes, afterSharders.Nodes, history,
		)
	})
}

func balanceSharderRewards(
	t *test.SystemTest,
	startRound, endRound int64,
	sharderIds []string,
	beforeSharders, afterSharders []climodel.Node,
	history *cliutil.ChainHistory,
) {
	minerScConfig := getMinerScMap(t)
	numSharderDelegatesRewarded := int(minerScConfig["num_sharder_delegates_rewarded"])
	numShardersRewarded := int(minerScConfig["num_sharders_rewarded"])
	if numShardersRewarded > len(sharderIds) {
		numShardersRewarded = len(sharderIds)
	}
	if numShardersRewarded == 0 {
		return
	}
	require.EqualValues(t, startRound/int64(minerScConfig["epoch"]), endRound/int64(minerScConfig["epoch"]),
		"epoch changed during test, start %v finish %v",
		startRound/int64(minerScConfig["epoch"]), endRound/int64(minerScConfig["epoch"]))

	_, sharderBlockReward := blockRewards(startRound, minerScConfig)
	bwPerSharder := int64(float64(sharderBlockReward) / float64(numShardersRewarded))

	checkSharderBlockRewards(
		t,
		sharderIds,
		bwPerSharder,
		numSharderDelegatesRewarded,
		beforeSharders, afterSharders,
		history,
	)

	countSharderBlockRewards(
		t, startRound+1, endRound-1, numShardersRewarded, history,
	)

	countDelegatesRewarded(
		t, sharderIds, numSharderDelegatesRewarded, beforeSharders, history,
	)

	balanceSharderDelegatePoolBlockRewards(
		t, sharderIds, numSharderDelegatesRewarded, bwPerSharder, beforeSharders, afterSharders, history,
	)
}

// checkSharderBlockRewards
// The num_sharders_rewarded smart contract setting determines how many sharder
// we rewarded each round, or all sharders if less.
//
// The amount of the reward is a fraction of the block reward allocated to sharders each
// round. The fraction is the sharder's service charge. If the sharder has
// no stake pools then the reward becomes the full block reward.
//
// If a selected sharder has delegate pools, we reward num_sharder_delegates_rewarded
// of them proportionate with their balance, or all delegate pools if
// a sharder has less than num_sharder_delegates_rewarded

func checkSharderBlockRewards(
	t *test.SystemTest,
	sharderIds []string,
	bwPerSharder int64,
	numSharderDelegatesRewarded int,
	beforeSharders, afterSharders []climodel.Node,
	history *cliutil.ChainHistory,
) {
	for i, id := range sharderIds {
		var rewards int64
		for round := beforeSharders[i].RoundServiceChargeLastUpdated + 1; round <= afterSharders[i].RoundServiceChargeLastUpdated; round++ {
			roundHistory := history.RoundHistory(t, round)
			for _, pReward := range roundHistory.ProviderRewards {
				if pReward.ProviderId != id {
					continue
				}
				switch pReward.RewardType {
				case climodel.BlockRewardSharder:
					var expectedServiceCharge int64
					payAllToSharder := len(beforeSharders[i].StakePool.Pools) == 0 || numSharderDelegatesRewarded == 0
					if payAllToSharder {
						expectedServiceCharge = bwPerSharder
					} else {
						expectedServiceCharge = int64(float64(bwPerSharder) * beforeSharders[i].Settings.ServiceCharge)
					}
					require.InDeltaf(t, expectedServiceCharge, pReward.Amount, delta, "sharder service charge incorrect value on round %d", round)
					rewards += pReward.Amount
				case climodel.FeeRewardSharder:
					rewards += pReward.Amount
				default:
					require.Failf(t, "reward type %s not available to sharders", pReward.RewardType.String())
				}
			}
		}
		actualReward := afterSharders[i].Reward - beforeSharders[i].Reward
		if actualReward != rewards {
			require.InDeltaf(t, actualReward, rewards, delta,
				"rewards expected %v change in sharders reward during test %v", actualReward, rewards)
		}
	}
}

// countMinerBlockRewards
// Each round there should be exactly num_sharders_rewarded sharder block reward payment.
// We confirm that the count of rewarded sharders is correct.
func countSharderBlockRewards(
	t *test.SystemTest,
	start, end int64,
	numShardersRewarded int,
	history *cliutil.ChainHistory,
) {
	for round := start; round <= end; round++ {
		roundHistory := history.RoundHistory(t, round)
		shardersPaid := make(map[string]bool)
		for _, pReward := range roundHistory.ProviderRewards {
			if pReward.RewardType == climodel.BlockRewardSharder {
				_, found := shardersPaid[pReward.ProviderId]
				require.Falsef(t, found, "sharder %s receives more than one block reward on round %d", pReward.ProviderId, round)
				shardersPaid[pReward.ProviderId] = true
			}
		}
		require.Equal(t, numShardersRewarded, len(shardersPaid),
			"mismatch between expected count of sharders rewarded and actual number on round %d", round)
	}
}

// countDelegatesRewarded
// Each round each sharder rewarded should have num_sharder_delegates_rewarded of
// their delegates rewarded, or all delegates if less.
func countDelegatesRewarded(
	t *test.SystemTest,
	sharderIds []string,
	numSharderDelegatesRewarded int,
	beforeSharders []climodel.Node,
	history *cliutil.ChainHistory,
) {
	for round := history.From(); round <= history.To(); round++ {
		roundHistory := history.RoundHistory(t, round)
		for i, id := range sharderIds {
			poolsPaid := make(map[string]bool)
			for poolId := range beforeSharders[i].Pools {
				for _, dReward := range roundHistory.DelegateRewards {
					if dReward.RewardType != climodel.BlockRewardSharder || dReward.PoolID != poolId {
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
			if numShouldPay > len(beforeSharders[i].Pools) {
				numShouldPay = len(beforeSharders[i].Pools)
			}
			require.Len(t, poolsPaid, numShouldPay,
				"should pay %d pools for shader %s on round %d; %d pools actually paid",
				numShouldPay, id, round, len(poolsPaid))
		}
	}
}

// balanceSharderDelegatePoolBlockRewards
// Compare the actual change in rewards to each sharder delegate, with the
// change expected from the delegate reward table.
func balanceSharderDelegatePoolBlockRewards(
	t *test.SystemTest,
	sharderIds []string,
	numSharderDelegatesRewarded int,
	bwPerSharder int64,
	beforeSharders, afterSharders []climodel.Node,
	history *cliutil.ChainHistory,
) {
	for i, id := range sharderIds {
		delegateBlockReward := int64(float64(bwPerSharder) * (1 - beforeSharders[i].Settings.ServiceCharge))
		numPools := len(afterSharders[i].StakePool.Pools)
		rewards := make(map[string]int64, numPools)
		for poolId := range afterSharders[i].StakePool.Pools {
			rewards[poolId] = 0
		}
		for round := beforeSharders[i].RoundServiceChargeLastUpdated + 1; round <= afterSharders[i].RoundServiceChargeLastUpdated; round++ {
			poolsBlockRewards := make(map[string]int64)
			roundHistory := history.RoundHistory(t, round)
			for _, dReward := range roundHistory.DelegateRewards {
				if dReward.ProviderID != id {
					continue
				}
				_, isSharderPool := rewards[dReward.PoolID]
				require.Truef(t, isSharderPool, "round %d, invalid pool id, reward %v", round, dReward)
				switch dReward.RewardType {
				case climodel.BlockRewardSharder:
					_, found := poolsBlockRewards[dReward.PoolID]
					require.False(t, found, "pool %s gets more than one block reward on round %d",
						dReward.PoolID, round)
					poolsBlockRewards[dReward.PoolID] = dReward.Amount
					rewards[dReward.PoolID] += dReward.Amount
				case climodel.FeeRewardSharder:
					rewards[dReward.PoolID] += dReward.Amount
				default:
					require.Failf(t, "reward type %s not available to sharders stake pools;"+
						" received by sharder %s on round %d", dReward.RewardType.String(), &dReward.PoolID, round)
				}
			}
			confirmPoolPayments(
				t, delegateBlockReward, poolsBlockRewards, afterSharders[i].StakePool.Pools, numSharderDelegatesRewarded,
			)
		}
		for poolId := range afterSharders[i].StakePool.Pools {
			actualReward := afterSharders[i].StakePool.Pools[poolId].Reward - beforeSharders[i].StakePool.Pools[poolId].Reward
			require.InDeltaf(t, actualReward, rewards[poolId], delta,
				"rewards expected %v, change in rewards during test %v", actualReward, rewards[poolId])
		}
	}
}

func getSortedSharderIds(t *test.SystemTest, sharderBaseURL string) []string {
	return getSortedNodeIds(t, "getSharderList", sharderBaseURL)
}
