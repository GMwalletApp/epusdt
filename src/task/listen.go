package task

import (
	"github.com/assimon/luuu/util/log"
	"github.com/robfig/cron/v3"
)

func Start() {
	log.Sugar.Info("[task] Starting task scheduler...")
	go StartEthereumWebSocketListener()
	go StartBscWebSocketListener()
	go StartPolygonWebSocketListener()
	go StartPlasmaWebSocketListener()
	go StartTronTrc20WebSocketListener()

	c := cron.New()
	// solana钱包监听
	_, err := c.AddJob("@every 5s", ListenSolJob{})
	if err != nil {
		log.Sugar.Errorf("[task] Failed to add ListenSolJob: %v", err)
		return
	}
	log.Sugar.Info("[task] ListenSolJob scheduled successfully (@every 5s)")
	c.Start()
	log.Sugar.Info("[task] Task scheduler started")
}
