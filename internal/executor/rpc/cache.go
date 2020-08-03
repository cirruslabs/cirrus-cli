package rpc

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/cache"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
)

const bufSize = 10 * 1024 * 1024

func (r *RPC) UploadCache(stream api.CirrusCIService_UploadCacheServer) error {
	var putOp *cache.PutOperation
	var bytesSaved int64

	for {
		cacheEntry, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			r.logger.WithContext(stream.Context()).WithError(err).Warn("stream errored out while uploading cache")
			return err
		}

		switch x := cacheEntry.Value.(type) {
		case *api.CacheEntry_Key:
			if putOp != nil {
				r.logger.WithContext(stream.Context()).Warn("received multiple cache entries in a single method call")
				return status.Error(codes.FailedPrecondition, "received multiple cache entries in a single method call")
			}
			_, err := r.build.GetTaskFromIdentification(x.Key.TaskIdentification, r.clientSecret)
			if err != nil {
				return err
			}
			putOp, err = r.build.Cache.Put(x.Key.CacheKey)
			if err != nil {
				r.logger.WithContext(stream.Context()).WithError(err).Debug("while initializing cache put operation")
				return status.Error(codes.Internal, "failed to initialize cache put operation")
			}
			r.logger.WithContext(stream.Context()).Debugf("receiving cache with key %s", x.Key.CacheKey)
		case *api.CacheEntry_Chunk:
			if putOp == nil {
				return status.Error(codes.PermissionDenied, "not authenticated")
			}
			n, err := putOp.Write(x.Chunk.Data)
			if err != nil {
				r.logger.WithContext(stream.Context()).WithError(err).Debug("while processing cache chunk")
				return status.Error(codes.Internal, "failed to process cache chunk")
			}
			bytesSaved += int64(n)
		}
	}

	if putOp == nil {
		r.logger.Warn("attempted to upload cache without actually sending anything")
		return status.Error(codes.FailedPrecondition, "attempted to upload cache without actually sending anything")
	}

	if err := putOp.Finalize(); err != nil {
		r.logger.WithError(err).Debugf("while finalizing cache put operation")
		return status.Error(codes.Internal, "failed to finalize cache put operation")
	}

	response := api.UploadCacheResponse{
		BytesSaved: bytesSaved,
	}
	if err := stream.SendAndClose(&response); err != nil {
		r.logger.WithContext(stream.Context()).WithError(err).Warn("while closing cache upload stream")
		return err
	}

	return nil
}

func (r *RPC) DownloadCache(req *api.DownloadCacheRequest, stream api.CirrusCIService_DownloadCacheServer) error {
	_, err := r.build.GetTaskFromIdentification(req.TaskIdentification, r.clientSecret)
	if err != nil {
		return err
	}

	r.logger.WithContext(stream.Context()).Debugf("sending cache with key %s", req.CacheKey)

	file, err := r.build.Cache.Get(req.CacheKey)
	if err != nil {
		return status.Errorf(codes.NotFound, "cache blob with the specified key not found")
	}
	defer file.Close()

	buf := make([]byte, bufSize)

	for {
		n, err := file.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to read cache blob")
		}

		chunk := api.DataChunk{
			Data: buf[:n],
		}
		err = stream.Send(&chunk)
		if err == io.EOF {
			break
		}
		if err != nil {
			r.logger.WithContext(stream.Context()).WithError(err).Warn("while sending cache chunk")
			return err
		}
	}

	return nil
}

func (r *RPC) CacheInfo(ctx context.Context, req *api.CacheInfoRequest) (*api.CacheInfoResponse, error) {
	_, err := r.build.GetTaskFromIdentification(req.TaskIdentification, r.clientSecret)
	if err != nil {
		return nil, err
	}

	r.logger.Debugf("sending info about cache key %s", req.CacheKey)

	file, err := r.build.Cache.Get(req.CacheKey)
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
			Key:               req.CacheKey,
			SizeInBytes:       fileInfo.Size(),
			CreationTimestamp: fileInfo.ModTime().Unix(),
		},
	}

	return &response, nil
}
