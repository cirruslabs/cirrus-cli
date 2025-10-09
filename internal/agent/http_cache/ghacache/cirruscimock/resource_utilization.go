package cirruscimock

import (
	"context"

	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (mock *cirrusCIMock) ReportAgentResourceUtilization(
	ctx context.Context,
	request *api.ReportAgentResourceUtilizationRequest,
) (*emptypb.Empty, error) {
	mock.intermediateResourceUtilizations = append(mock.intermediateResourceUtilizations, request.GetResourceUtilization())

	return &emptypb.Empty{}, nil
}

func (mock *cirrusCIMock) InspectIntermediateResourceUtilizations() []*api.ResourceUtilization {
	return mock.intermediateResourceUtilizations
}
