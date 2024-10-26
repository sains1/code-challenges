package cmd

import (
	"os"

	"github.com/codecrafters-io/redis-starter-go/app/rdb"
	"github.com/codecrafters-io/redis-starter-go/app/replication"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
	"github.com/google/uuid"
)

func HandlePSync(ctx HandleContext) {
	// TODO hardcoded response for this stage only
	ctx.Conn.Write([]byte(resp.PSyncResponse(ctx.HostCtx.LeaderReplId, 0).AsRespString()))

	rdbfile := rdb.ReadRdb()
	rdbstr, err := rdb.SerializeB64RdbToString(rdbfile)

	if err != nil {
		ctx.Logger.Error().Err(err).Msg("Error serializing rdb file")
		os.Exit(1)
	}

	ctx.Conn.Write([]byte(rdbstr))

	subscriberId := uuid.New().String()

	if err != nil {
		ctx.Logger.Error().Err(err).Msg("Error generating subscription id")
		os.Exit(1)
	}

	// begin replicating to the replica
	replicationChannel := make(chan replication.PubSubEvent)

	ctx.HostCtx.PubSubManager.SubscriptionsChannel <- replication.SubscriberEvent{
		Action:            replication.SubscribeAction,
		SubscriberId:      subscriberId,
		SubscriberChannel: replicationChannel,
	}

	defer func() {
		ctx.HostCtx.PubSubManager.SubscriptionsChannel <- replication.SubscriberEvent{
			Action:       replication.UnsubscribeAction,
			SubscriberId: subscriberId,
		}

		close(replicationChannel)
	}()

	for event := range replicationChannel {
		_, err := ctx.Conn.Write([]byte(event))

		if err != nil {
			ctx.Logger.Error().Err(err).Msg("Failed to write to follower, connection may be closed")
			break // Exit the loop if the connection is closed
		}
	}
}
