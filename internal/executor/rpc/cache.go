package rpc

import (
	"context"
	"os"

	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/samber/lo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (r *RPC) CacheInfo(ctx context.Context, req *api.CacheInfoRequest) (*api.CacheInfoResponse, error) {
	_, err := r.build.GetTaskFromIdentification(req.TaskIdentification, r.clientSecret)
	if err != nil {
		return nil, err
	}

	r.logger.Debugf("sending info about cache key %s", req.CacheKey)

	var file *os.File

	var prefixMatch bool

	if req.CacheKey != "" {
		file, err = r.build.Cache.Get(req.CacheKey)
	}
	if file == nil {
		for _, prefix := range req.CacheKeyPrefixes {
			file, err = r.build.Cache.FindByPrefix(prefix)
			if file != nil {
				prefixMatch = true

				break
			}
		}
	}
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "cache blob with the specified key not found")
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	response := api.CacheInfoResponse{
		Info: &api.CacheInfo{
			Key:               lo.Ternary(prefixMatch, fileInfo.Name(), req.CacheKey),
			SizeInBytes:       fileInfo.Size(),
			CreationTimestamp: fileInfo.ModTime().Unix(),
		},
	}

	return &response, nil
}
