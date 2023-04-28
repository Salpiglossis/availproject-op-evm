package tests

import (
	"context"
	"flag"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/maticnetwork/avail-settlement/consensus/avail"
)

// nolint:unused
var availAddr = flag.String("avail-addr", "ws://127.0.0.1:9944/v1/json-rpc", "Avail JSON-RPC URL")

// nolint:unused
var bindInterface = flag.String("bind-addr", "127.0.0.1", "IP address of the interface to bind node ports to")

// nolint:unused
var genesisCfgPath = flag.String("genesis-config", "../configs/genesis.json", "Path to genesis.json config")

// nolint:unused
var accountPath = flag.String("account-config-file", "../configs/account", "Path to the account mnemonic file")

var awsInstancesFlag = flag.String("aws-instances", "", "file containing all the information about the aws instances deployed as json, if provided will be used to connect instead of spawning up own instances")

func Test_MultipleSequencers(t *testing.T) {
	// t.Skip("multi-sequencer e2e tests disabled in CI/CD due to lack of Avail")

	flag.Parse()
	runtime.GOMAXPROCS(32)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	*genesisCfgPath = filepath.Join(cwd, *genesisCfgPath)

	var ctx ContextInterface
	if *awsInstancesFlag == "" {
		t.Log("starting nodes")
		bindAddr, err := netip.ParseAddr(*bindInterface)
		if err != nil {
			t.Fatal(err)
		}

		// ctx, err = StartNodes(t, bindAddr, *genesisCfgPath, *availAddr, *accountPath, avail.BootstrapSequencer, avail.Sequencer, avail.Sequencer, avail.WatchTower)
		ctx, err = StartNodes(t, bindAddr, *genesisCfgPath, *availAddr, *accountPath, avail.BootstrapSequencer, avail.Sequencer) //, avail.Sequencer, avail.Sequencer)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		ctx, err = NewDevnetContext(*awsInstancesFlag)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Shutdown all nodes once test finishes.
	t.Cleanup(ctx.StopAll)

	t.Log("nodes started")

	ethClient, err := ctx.GethClient()
	if err != nil {
		t.Fatal(err)
	}

	waitForPeers(t, ethClient, 1)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		bNum, err := ethClient.BlockNumber(ctx)
		if err != nil {
			t.Fatal(err)
		}

		// Cleanup timeout context.
		cancel()

		// Wait for 5 blocks
		if bNum > 49 {
			break
		}

		time.Sleep(time.Second)
	}
}

// nolint:unused
func waitForPeers(t testing.TB, ethClient *ethclient.Client, minNodes int) {
	t.Helper()

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

		peerCount, err := ethClient.PeerCount(ctx)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("Got peer count: %d", peerCount)

		// Cleanup timeout context.
		cancel()

		if int(peerCount) >= minNodes {
			return
		}

		time.Sleep(250 * time.Millisecond)
	}
}
