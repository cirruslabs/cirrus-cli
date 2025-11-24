package executor

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/internal/agent/environment"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

type LogUploader struct {
	taskIdentification *api.TaskIdentification
	commandName        string
	client             api.CirrusCIService_StreamLogsClient
	storedOutput       *os.File
	erroredChunks      int
	logsChannel        chan []byte
	doneLogUpload      chan bool
	env                *environment.Environment
	closed             bool

	// Fields related to the CIRRUS_LOG_TIMESTAMP behavioral environment variable
	LogTimestamps bool
	GetTimestamp  func() time.Time
	OweTimestamp  bool

	mutex sync.RWMutex
}

func NewLogUploader(ctx context.Context, executor *Executor, commandName string) (*LogUploader, error) {
	logClient, err := InitializeLogStreamClient(ctx, executor.taskIdentification(), commandName, false)
	if err != nil {
		return nil, err
	}
	EnsureFolderExists(os.TempDir())
	file, err := os.CreateTemp(os.TempDir(), commandName)
	if err != nil {
		return nil, err
	}
	logUploader := LogUploader{
		taskIdentification: executor.taskIdentification(),
		commandName:        commandName,
		client:             logClient,
		storedOutput:       file,
		erroredChunks:      0,
		logsChannel:        make(chan []byte, 128),
		doneLogUpload:      make(chan bool),
		env:                executor.env,
		closed:             false,

		LogTimestamps: executor.env.Get("CIRRUS_LOG_TIMESTAMP") == "true",
		GetTimestamp:  time.Now,
		OweTimestamp:  true,
	}
	go logUploader.StreamLogs()
	return &logUploader, nil
}

func (uploader *LogUploader) reInitializeClient(ctx context.Context) error {
	err := uploader.client.CloseSend()
	if err != nil {
		slog.Warn("Failed to close log for reinitialization", "command", uploader.commandName, "err", err)
	}
	logClient, err := InitializeLogStreamClient(ctx, uploader.taskIdentification, uploader.commandName, false)
	if err != nil {
		return err
	}
	uploader.client = logClient
	return nil
}

func (uploader *LogUploader) WithTimestamps(input []byte) []byte {
	var result []byte

	timestampPrefix := uploader.GetTimestamp().Format("[15:04:05.000]") + " "

	// Insert a timestamp if we owe one, either because it's
	// the first log chunk in the stream or because the previous
	// chunk was ending with \n
	if uploader.OweTimestamp {
		result = append(result, []byte(timestampPrefix)...)
		uploader.OweTimestamp = false
	}

	// How many \n's are going to get a timestamp prefix
	numTimestamps := bytes.Count(input, []byte{'\n'})

	// If the chunk ends with \n — don't insert the timestamp at the end
	// right now, but remember to do this in the future to avoid empty
	// lines with timestamps at the log's end
	if bytes.HasSuffix(input, []byte{'\n'}) {
		numTimestamps--
		uploader.OweTimestamp = true
	}

	// Insert timestamps
	input = bytes.Replace(input, []byte("\n"), []byte("\n"+timestampPrefix), numTimestamps)
	result = append(result, input...)

	return result
}

func (uploader *LogUploader) Write(bytes []byte) (int, error) {
	if len(bytes) == 0 {
		return 0, nil
	}

	// Make potential bytes expansion below transparent to the caller
	originalLen := len(bytes)

	if uploader.LogTimestamps {
		bytes = uploader.WithTimestamps(bytes)
	}

	uploader.mutex.RLock()
	defer uploader.mutex.RUnlock()
	if !uploader.closed {
		bytesCopy := make([]byte, len(bytes))
		copy(bytesCopy, bytes)
		uploader.logsChannel <- bytesCopy
	}
	return originalLen, nil
}

func (uploader *LogUploader) StreamLogs() {
	ctx := context.Background()

	for {
		logs, finished := uploader.ReadAvailableChunks()
		_, err := uploader.WriteChunk(logs)
		if finished {
			slog.Info("Finished streaming logs", "command", uploader.commandName)
			break
		}
		if err == io.EOF {
			slog.Warn("Got EOF while streaming logs, trying to reinitialize", "command", uploader.commandName)
			err := uploader.reInitializeClient(ctx)
			if err == nil {
				slog.Info("Successfully reinitialized log uploader", "command", uploader.commandName)
			} else {
				slog.Error("Failed to reinitialize log uploader", "command", uploader.commandName, "err", err)
			}
		}
	}
	uploader.client.CloseAndRecv()

	err := uploader.UploadStoredOutput(ctx)
	if err != nil {
		slog.Error("Failed to upload stored logs", "command", uploader.commandName, "err", err)
	} else {
		slog.Info("Uploaded stored logs", "command", uploader.commandName)
	}

	uploader.storedOutput.Close()
	os.Remove(uploader.storedOutput.Name())

	uploader.doneLogUpload <- true
}

func (uploader *LogUploader) ReadAvailableChunks() ([]byte, bool) {
	const maxBytesPerInvocation = 1 * 1024 * 1024

	// Make sure we wait first to avoid busy loop in StreamLogs()
	result := <-uploader.logsChannel

	// Read log chunks from the channel, but no more than maxBytesPerInvocation bytes
	//
	// This assumes that log chunks are small by themselves (e.g. 32,000 bytes).
	for {
		select {
		case nextChunk, more := <-uploader.logsChannel:
			result = append(result, nextChunk...)
			if !more {
				slog.Info("No more log chunks", "command", uploader.commandName)
				return result, true
			}
		default:
			return result, false
		}

		if len(result) > maxBytesPerInvocation {
			return result, false
		}
	}
}

func (uploader *LogUploader) WriteChunk(bytesToWrite []byte) (int, error) {
	if len(bytesToWrite) == 0 {
		return 0, nil
	}
	for _, valueToMask := range uploader.env.SensitiveValues() {
		bytesToWrite = bytes.Replace(bytesToWrite, []byte(valueToMask), []byte("HIDDEN-BY-CIRRUS-CI"), -1)
	}

	uploader.storedOutput.Write(bytesToWrite)
	dataChunk := api.DataChunk{Data: bytesToWrite}
	logEntry := api.LogEntry_Chunk{Chunk: &dataChunk}
	err := uploader.client.Send(&api.LogEntry{Value: &logEntry})
	if err != nil {
		slog.Error("Failed to send logs", "command", uploader.commandName, "err", err, "bytes", string(bytesToWrite))
		uploader.erroredChunks++
		return 0, err
	}
	return len(bytesToWrite), nil
}

func (uploader *LogUploader) Finalize() {
	slog.Info("Finalizing log uploading", "command", uploader.commandName)
	uploader.mutex.Lock()
	uploader.closed = true
	close(uploader.logsChannel)
	uploader.mutex.Unlock()
	<-uploader.doneLogUpload
}

func (uploader *LogUploader) UploadStoredOutput(ctx context.Context) error {
	logClient, err := InitializeLogSaveClient(ctx, uploader.taskIdentification, uploader.commandName, true)
	if err != nil {
		return err
	}
	defer logClient.CloseAndRecv()

	if uploader.commandName == "test_unexpected_error_during_log_streaming" {
		dataChunk := api.DataChunk{Data: []byte("Live streaming of logs failed!\n")}
		logEntry := api.LogEntry_Chunk{Chunk: &dataChunk}
		err = logClient.Send(&api.LogEntry{Value: &logEntry})
		if err != nil {
			return err
		}
	}

	uploader.storedOutput.Seek(0, io.SeekStart)

	readBufferSize := int(1024 * 1024)
	readBuffer := make([]byte, readBufferSize)
	bufferedReader := bufio.NewReaderSize(uploader.storedOutput, readBufferSize)
	for {
		n, err := bufferedReader.Read(readBuffer)

		if n > 0 {
			dataChunk := api.DataChunk{Data: readBuffer[:n]}
			logEntry := api.LogEntry_Chunk{Chunk: &dataChunk}
			err = logClient.Send(&api.LogEntry{Value: &logEntry})
		}

		if err == io.EOF || n == 0 {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func InitializeLogStreamClient(ctx context.Context, taskIdentification *api.TaskIdentification, commandName string, raw bool) (api.CirrusCIService_StreamLogsClient, error) {
	var streamLogClient api.CirrusCIService_StreamLogsClient
	var err error

	err = retry.Do(func() error {
		streamLogClient, err = client.CirrusClient.StreamLogs(ctx, grpc.UseCompressor(gzip.Name))
		return err
	}, retry.Delay(5*time.Second), retry.Attempts(3), retry.Context(ctx))
	if err != nil {
		slog.Error("Failed to start streaming logs", "command", commandName, "err", err)
		request := api.ReportAgentProblemRequest{
			TaskIdentification: taskIdentification,
			Message:            fmt.Sprintf("Failed to start streaming logs for command %v: %v", commandName, err),
		}
		client.CirrusClient.ReportAgentWarning(ctx, &request)
		return nil, err
	}
	logEntryKey := api.LogEntry_LogKey{TaskIdentification: taskIdentification, CommandName: commandName, Raw: raw}
	logEntry := api.LogEntry_Key{Key: &logEntryKey}
	streamLogClient.Send(&api.LogEntry{Value: &logEntry})
	return streamLogClient, nil
}

func InitializeLogSaveClient(
	ctx context.Context,
	taskIdentification *api.TaskIdentification,
	commandName string,
	raw bool,
) (api.CirrusCIService_SaveLogsClient, error) {
	var streamLogClient api.CirrusCIService_StreamLogsClient
	var err error

	err = retry.Do(
		func() error {
			streamLogClient, err = client.CirrusClient.SaveLogs(ctx, grpc.UseCompressor(gzip.Name))
			return err
		},
		retry.Delay(5*time.Second),
		retry.Attempts(3),
	)
	if err != nil {
		slog.Error("Failed to start saving logs", "command", commandName, "err", err)
		request := api.ReportAgentProblemRequest{
			TaskIdentification: taskIdentification,
			Message:            fmt.Sprintf("Failed to start saving logs for command %v: %v", commandName, err),
		}
		client.CirrusClient.ReportAgentWarning(ctx, &request)
		return nil, err
	}
	logEntryKey := api.LogEntry_LogKey{TaskIdentification: taskIdentification, CommandName: commandName, Raw: raw}
	logEntry := api.LogEntry_Key{Key: &logEntryKey}
	streamLogClient.Send(&api.LogEntry{Value: &logEntry})
	return streamLogClient, nil
}
