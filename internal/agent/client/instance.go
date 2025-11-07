package client

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/blobstorage"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"google.golang.org/grpc"
)

var CirrusClient api.CirrusCIServiceClient
var CirrusTaskIdentification *api.TaskIdentification

func InitClient(conn *grpc.ClientConn, taskId string, clientToken string) blobstorage.BlobStorageBacked {
	CirrusClient = api.NewCirrusCIServiceClient(conn)
	CirrusTaskIdentification = api.OldTaskIdentification(taskId, clientToken)
	return NewCirrusBlobStorage(CirrusClient)
}
