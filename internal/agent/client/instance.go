package client

import (
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"google.golang.org/grpc"
)

var CirrusClient api.CirrusCIServiceClient
var CirrusTaskIdentification *api.TaskIdentification

func InitClient(conn *grpc.ClientConn, taskIdentification *api.TaskIdentification) {
	CirrusClient = api.NewCirrusCIServiceClient(conn)
	CirrusTaskIdentification = taskIdentification
}
