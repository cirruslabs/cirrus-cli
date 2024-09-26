package client

import (
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"google.golang.org/grpc"
)

var CirrusClient api.CirrusCIServiceClient
var CirrusTaskIdentification *api.TaskIdentification

func InitClient(conn *grpc.ClientConn, taskId string, clientToken string) {
	CirrusClient = api.NewCirrusCIServiceClient(conn)
	CirrusTaskIdentification = api.OldTaskIdentification(taskId, clientToken)
}
