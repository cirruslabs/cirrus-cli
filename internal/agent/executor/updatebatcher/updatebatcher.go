package updatebatcher

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"log"
)

type UpdateBatcher struct {
	updateHistory    []*api.CommandResult
	unflushedUpdates []*api.CommandResult
}

func New() *UpdateBatcher {
	return &UpdateBatcher{
		updateHistory:    []*api.CommandResult{},
		unflushedUpdates: []*api.CommandResult{},
	}
}

func (ub *UpdateBatcher) Queue(update *api.CommandResult) {
	ub.updateHistory = append(ub.updateHistory, update)
	ub.unflushedUpdates = append(ub.unflushedUpdates, update)
}

func (ub *UpdateBatcher) Flush(ctx context.Context, taskIdentification *api.TaskIdentification) {
	if len(ub.unflushedUpdates) == 0 {
		return
	}

	_, err := client.CirrusClient.ReportCommandUpdates(ctx, &api.ReportCommandUpdatesRequest{
		TaskIdentification: taskIdentification,
		Updates:            ub.unflushedUpdates,
	})
	if err != nil {
		log.Printf("Failed to report command updates: %v\n", err)
		return
	}
	ub.unflushedUpdates = ub.unflushedUpdates[:0]
}

func (ub *UpdateBatcher) History() []*api.CommandResult {
	return ub.updateHistory
}
