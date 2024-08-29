package cirruscimock

import (
	"bytes"
	"context"
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"io"
	"net"
	"testing"
	"time"
)

type cirrusCIMock struct {
	s3Client *s3.S3
	s3Bucket *string

	api.UnimplementedCirrusCIServiceServer
}

func newCirrusCIMock(t *testing.T, s3Client *s3.S3) *cirrusCIMock {
	mock := &cirrusCIMock{
		s3Client: s3Client,
		s3Bucket: aws.String("test"),
	}

	_, err := mock.s3Client.CreateBucket(&s3.CreateBucketInput{
		Bucket: mock.s3Bucket,
	})
	require.NoError(t, err)

	return mock
}

func ClientConn(t *testing.T) *grpc.ClientConn {
	t.Helper()

	s3Client := S3Client(t)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		server := grpc.NewServer()
		api.RegisterCirrusCIServiceServer(server, newCirrusCIMock(t, s3Client))
		require.NoError(t, server.Serve(lis))
	}()

	clientConn, err := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	return clientConn
}

func (mock *cirrusCIMock) UploadCache(stream api.CirrusCIService_UploadCacheServer) error {
	cacheEntries := map[string]*bytes.Buffer{}
	var currentBuf *bytes.Buffer

	for {
		cacheEntry, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return err
		}

		switch typed := cacheEntry.Value.(type) {
		case *api.CacheEntry_Key:
			currentBuf = &bytes.Buffer{}
			cacheEntries[typed.Key.CacheKey] = currentBuf
		case *api.CacheEntry_Chunk:
			currentBuf.Write(typed.Chunk.Data)
		}
	}

	for key, buf := range cacheEntries {
		_, err := mock.s3Client.PutObjectWithContext(stream.Context(), &s3.PutObjectInput{
			Bucket: mock.s3Bucket,
			Key:    aws.String(key),
			Body:   bytes.NewReader(buf.Bytes()),
		})
		if err != nil {
			return err
		}
	}

	return stream.SendAndClose(&api.UploadCacheResponse{})
}

func (mock *cirrusCIMock) DownloadCache(request *api.DownloadCacheRequest, stream api.CirrusCIService_DownloadCacheServer) error {
	result, err := mock.s3Client.GetObjectWithContext(stream.Context(), &s3.GetObjectInput{
		Bucket: mock.s3Bucket,
		Key:    aws.String(request.CacheKey),
	})
	if err != nil {
		var aerr awserr.Error
		if errors.As(err, &aerr) && aerr.Code() == s3.ErrCodeNoSuchKey {
			return status.Errorf(codes.NotFound, "cache entry for key %s is not found",
				request.CacheKey)
		}

		return err
	}
	defer result.Body.Close()

	buf, err := io.ReadAll(result.Body)
	if err != nil {
		return err
	}

	// Chunk the buffer to prevent the "grpc: received message larger than
	// max (X vs. 4194304)" error
	for _, chunk := range lo.Chunk(buf, 1*1024*1024) {
		if err := stream.Send(&api.DataChunk{Data: chunk}); err != nil {
			return err
		}
	}

	return nil
}

func (mock *cirrusCIMock) CacheInfo(ctx context.Context, request *api.CacheInfoRequest) (*api.CacheInfoResponse, error) {
	result, err := mock.s3Client.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: mock.s3Bucket,
		Key:    aws.String(request.CacheKey),
	})
	if err != nil {
		var aerr awserr.Error
		if errors.As(err, &aerr) && aerr.Code() == s3.ErrCodeNoSuchKey {
			return nil, status.Errorf(codes.NotFound, "cache entry for key %s is not found",
				request.CacheKey)
		}

		return nil, err
	}

	return &api.CacheInfoResponse{
		Info: &api.CacheInfo{
			Key:         request.CacheKey,
			SizeInBytes: *result.ContentLength,
		},
	}, nil
}

func (mock *cirrusCIMock) MultipartCacheUploadCreate(ctx context.Context, request *api.CacheKey) (*api.MultipartCacheUploadCreateResponse, error) {
	result, err := mock.s3Client.CreateMultipartUploadWithContext(ctx, &s3.CreateMultipartUploadInput{
		Bucket: mock.s3Bucket,
		Key:    aws.String(request.CacheKey),
	})
	if err != nil {
		return nil, err
	}

	return &api.MultipartCacheUploadCreateResponse{
		UploadId: *result.UploadId,
	}, nil
}

func (mock *cirrusCIMock) MultipartCacheUploadPart(ctx context.Context, request *api.MultipartCacheUploadPartRequest) (*api.GenerateURLResponse, error) {
	uploadPartRequest, _ := mock.s3Client.UploadPartRequest(&s3.UploadPartInput{
		Bucket:     mock.s3Bucket,
		Key:        aws.String(request.CacheKey.CacheKey),
		UploadId:   aws.String(request.UploadId),
		PartNumber: aws.Int64(int64(request.PartNumber)),
	})

	url, headers, err := uploadPartRequest.PresignRequest(10 * time.Minute)
	if err != nil {
		return nil, err
	}

	return &api.GenerateURLResponse{
		Url: url,
		ExtraHeaders: lo.MapEntries(headers, func(key string, value []string) (string, string) {
			return key, value[0]
		}),
	}, nil
}

func (mock *cirrusCIMock) MultipartCacheUploadCommit(ctx context.Context, request *api.MultipartCacheUploadCommitRequest) (*emptypb.Empty, error) {
	var parts []*s3.CompletedPart

	for _, part := range request.Parts {
		parts = append(parts, &s3.CompletedPart{
			PartNumber: aws.Int64(int64(part.PartNumber)),
			ETag:       aws.String(part.Etag),
		})
	}

	_, err := mock.s3Client.CompleteMultipartUploadWithContext(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   mock.s3Bucket,
		Key:      aws.String(request.CacheKey.CacheKey),
		UploadId: aws.String(request.UploadId),
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: parts,
		},
	})
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
