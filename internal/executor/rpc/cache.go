package rpc

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/executor/cache"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/samber/lo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"os"
)

const sendBufSize = 1024 * 1024

func (r *RPC) UploadCache(stream api.CirrusCIService_UploadCacheServer) error {
	if _, err := r.taskFromMetadata(stream.Context()); err != nil {
		return err
	}

	var putOp *cache.PutOperation
	var bytesSaved int64

	for {
		cacheEntry, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			r.logger.Warnf("error stream errored out while uploading cache: %v", err)
			return err
		}

		switch x := cacheEntry.Value.(type) {
		case *api.CacheEntry_Key:
			if putOp != nil {
				r.logger.Warnf("received multiple cache entries in a single method call")
				return status.Error(codes.FailedPrecondition, "received multiple cache entries in a single method call")
			}
			putOp, err = r.build.Cache.Put(x.Key.CacheKey)
			if err != nil {
				r.logger.Debugf("error while initializing cache put operation: %v", err)
				return status.Error(codes.Internal, "failed to initialize cache put operation")
			}
			r.logger.Debugf("receiving cache with key %s", x.Key.CacheKey)
		case *api.CacheEntry_Chunk:
			if putOp == nil {
				return status.Error(codes.PermissionDenied, "not authenticated")
			}
			n, err := putOp.Write(x.Chunk.Data)
			if err != nil {
				r.logger.Debugf("error while processing cache chunk: %v", err)
				return status.Error(codes.Internal, "failed to process cache chunk")
			}
			bytesSaved += int64(n)
		}
	}

	if putOp == nil {
		r.logger.Warnf("attempted to upload cache without actually sending anything")
		return status.Error(codes.FailedPrecondition, "attempted to upload cache without actually sending anything")
	}

	if err := putOp.Finalize(); err != nil {
		r.logger.Debugf("while finalizing cache put operation")
		return status.Error(codes.Internal, "failed to finalize cache put operation")
	}

	response := api.UploadCacheResponse{
		BytesSaved: bytesSaved,
	}
	if err := stream.SendAndClose(&response); err != nil {
		r.logger.Warnf("error while closing cache upload stream: %v", err)
		return err
	}

	return nil
}

func (r *RPC) DownloadCache(req *api.DownloadCacheRequest, stream api.CirrusCIService_DownloadCacheServer) error {
	if _, err := r.taskFromMetadata(stream.Context()); err != nil {
		return err
	}

	file, err := r.build.Cache.Get(req.CacheKey)
	if err != nil {
		r.logger.Debugf("error while getting cache blob with key %s: %v", req.CacheKey, err)
		return status.Errorf(codes.NotFound, "cache blob with the specified key not found")
	}
	defer file.Close()

	r.logger.Debugf("sending cache with key %s", req.CacheKey)

	buf := make([]byte, sendBufSize)

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
			r.logger.Warnf("error while sending cache chunk of size %d: %v", n, err)
			return err
		}
	}

	return nil
}

func (r *RPC) CacheInfo(ctx context.Context, req *api.CacheInfoRequest) (*api.CacheInfoResponse, error) {
	if _, err := r.taskFromMetadata(ctx); err != nil {
		return nil, err
	}

	r.logger.Debugf("sending info about cache key %s", req.CacheKey)

	var file *os.File
	var err error

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
