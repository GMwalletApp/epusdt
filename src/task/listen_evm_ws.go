package task

import (
	"context"
	"time"

	"github.com/assimon/luuu/util/log"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func runWsLogListener(logPrefix, wsURL string, query ethereum.FilterQuery, handleLog func(*ethclient.Client, types.Log)) {
	const (
		minBackoff = 2 * time.Second
		maxBackoff = 60 * time.Second
		rejoinWait = 3 * time.Second
	)
	failWait := minBackoff

	for {
		client, err := ethclient.Dial(wsURL)
		if err != nil {
			log.Sugar.Warnf("%s dial: %v, retry in %s", logPrefix, err, failWait)
			time.Sleep(failWait)
			failWait = nextBackoff(failWait, maxBackoff)
			continue
		}

		logsCh := make(chan types.Log)
		sub, err := client.SubscribeFilterLogs(context.Background(), query, logsCh)
		if err != nil {
			client.Close()
			log.Sugar.Warnf("%s subscribe: %v, retry in %s", logPrefix, err, failWait)
			time.Sleep(failWait)
			failWait = nextBackoff(failWait, maxBackoff)
			continue
		}
		failWait = minBackoff

		log.Sugar.Infof("%s connected, subscribed to USDT/USDC Transfer logs", logPrefix)

		recvLoop(client, sub, logsCh, logPrefix, handleLog)

		time.Sleep(rejoinWait)
	}
}

func recvLoop(client *ethclient.Client, sub ethereum.Subscription, logsCh <-chan types.Log, logPrefix string, handleLog func(*ethclient.Client, types.Log)) {
	defer func() {
		sub.Unsubscribe()
		client.Close()
	}()

	for {
		select {
		case err := <-sub.Err():
			if err != nil {
				log.Sugar.Warnf("%s subscription error: %v, reconnecting", logPrefix, err)
			} else {
				log.Sugar.Warnf("%s subscription closed, reconnecting", logPrefix)
			}
			return
		case vLog, ok := <-logsCh:
			if !ok {
				log.Sugar.Warnf("%s log channel closed, reconnecting", logPrefix)
				return
			}
			handleLog(client, vLog)
		}
	}
}

func nextBackoff(cur, max time.Duration) time.Duration {
	n := cur * 2
	if n > max {
		return max
	}
	return n
}
