package rpc

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/golang/protobuf/ptypes/empty"
	"io"
)

func (r *RPC) SaveLogs(stream api.CirrusCIService_SaveLogsServer) error {
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			r.logger.Warnf("error while receiving saved logs: %v", err)
			return err
		}
	}

	if err := stream.SendAndClose(&api.UploadLogsResponse{}); err != nil {
		r.logger.Warnf("error while closing saved logs stream: %v", err)
		return err
	}

	return nil
}

func (r *RPC) ReportAgentLogs(ctx context.Context, req *api.ReportAgentLogsRequest) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}
