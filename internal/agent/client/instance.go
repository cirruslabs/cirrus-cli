package client

import (
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"google.golang.org/grpc"
)

var CirrusClient api.CirrusCIServiceClient

func InitClient(conn *grpc.ClientConn) {
	CirrusClient = api.NewCirrusCIServiceClient(conn)
}
