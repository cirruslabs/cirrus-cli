package executor

import (
	"bufio"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/bmatcuk/doublestar"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/internal/agent/environment"
	"github.com/cirruslabs/cirrus-cli/internal/agent/hasher"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache"
	"github.com/cirruslabs/cirrus-cli/internal/agent/targz"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Cache struct {
	Name                     string
	Key                      string
	BaseFolder               string
	PartiallyExpandedFolders []string
	FileHasher               *hasher.Hasher
	SkipUpload               bool
	CacheAvailable           bool
}

var caches = make([]Cache, 0)

var httpClient = &http.Client{
	Timeout: 10 * time.Minute,
}

func (executor *Executor) DownloadCache(
	ctx context.Context,
	logUploader *LogUploader,
	commandName string,
	cacheHost string,
	instruction *api.CacheInstruction,
	custom_env *environment.Environment,
) bool {
	cacheKey, ok := executor.generateCacheKey(ctx, logUploader, commandName, instruction, custom_env)
	if !ok {
		return false
	}

	// Partially expand cache folders without and keep them for further re-evaluation in UploadCache()
	//
	// Once in UploadCache(), the cache will be populated, and the globbing may yield a different result.
	var partiallyExpandedFolders []string

	for _, folder := range instruction.Folders {
		expandedFolder := custom_env.ExpandText(folder)

		absFolder, err := filepath.Abs(expandedFolder)
		// check if `getwd: no such file or directory`
		var syscallError *os.SyscallError
		if errors.As(err, &syscallError) && syscallError.Syscall == "getwd" && !filepath.IsAbs(expandedFolder) {
			logUploader.Write([]byte("\nFailed to get process working directory. Assuming CIRRUS_WORKING_DIR\n"))
			absFolder = path.Join(custom_env.Get("CIRRUS_WORKING_DIR"), expandedFolder)
		} else if err != nil {
			message := fmt.Sprintf("\nFailed to compute absolute path for cache folder '%s': %v\n", folder, err)
			executor.cacheAttempts.Failed(cacheKey, message)
			logUploader.Write([]byte(message))
			return false
		}

		partiallyExpandedFolders = append(partiallyExpandedFolders, absFolder)
	}

	// Determine the base folder
	baseFolder := custom_env.Get("CIRRUS_WORKING_DIR")
	if len(partiallyExpandedFolders) == 1 && !pathLooksLikeGlob(partiallyExpandedFolders[0]) {
		baseFolder = partiallyExpandedFolders[0]
	}

	// Perform a sanity check against the base folder
	//
	// When we're dealing with multiple cache folders, the semantics is
	// clearly defined only when all folders are scoped to the current
	// working directory, otherwise it's impossible to make paths inside
	// of the archive portable (i.e. independent on the location of the
	// working directory).
	//
	// Note: this is not a security stop-gap but merely a hint to the users
	// that they are doing something wrong.
	if len(partiallyExpandedFolders) > 1 {
		for _, partiallyExpandedFolder := range partiallyExpandedFolders {
			terminatedWorkingDir := baseFolder

			if !strings.HasSuffix(terminatedWorkingDir, string(os.PathSeparator)) {
				terminatedWorkingDir += string(os.PathSeparator)
			}

			if !strings.HasPrefix(partiallyExpandedFolder, terminatedWorkingDir) {
				message := fmt.Sprintf("\nWhen using globs or multiple cache folders, all folders should be relative to "+
					"the current working directory, yet, folder '%s' points above the current working directory '%s'\n",
					partiallyExpandedFolder, terminatedWorkingDir)
				executor.cacheAttempts.Failed(cacheKey, message)
				logUploader.Write([]byte(message))
				return false
			}
		}
	}

	cachePopulated, cacheAvailable := executor.tryToDownloadAndPopulateCache(ctx, logUploader, commandName, cacheHost, cacheKey, baseFolder)

	if !cacheAvailable && instruction.OptimisticallyRestoreOnMiss {
		logUploader.Write([]byte("\nWasn't able to find the exact cache! Requesting the last available one..."))
		cacheInfo := executor.findLatestAvailableCache(ctx, logUploader, commandName)
		if cacheInfo != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFound cache entry %s created by task %s!", cacheInfo.Key, cacheInfo.CreatedByTaskId)))
			cachePopulated, cacheAvailable = executor.tryToDownloadAndPopulateCache(ctx, logUploader, commandName, cacheHost, cacheInfo.Key, baseFolder)
		}
	}

	// Expand cache folders in case they contain potential globs,
	// so we can calculate the hashes for directories that already exist
	foldersToCache, message := executor.expandAndDeduplicateGlobs(partiallyExpandedFolders)
	if message != "" {
		executor.cacheAttempts.Failed(cacheKey, message)
		logUploader.Write([]byte(message))
		return false
	}

	fileHasher := hasher.New()
	if cachePopulated {
		for _, folderToCache := range foldersToCache {
			if err := fileHasher.AddFolder(baseFolder, folderToCache); err != nil {
				logUploader.Write([]byte(fmt.Sprintf("\nFailed to calculate hash of %s! %s", folderToCache, err)))
			}
		}
	}

	if !cachePopulated && len(instruction.PopulateScripts) > 0 {
		populateStartTime := time.Now()
		logUploader.Write([]byte(fmt.Sprintf("\nCache miss for %s! Populating...\n", cacheKey)))
		cmd, err := ShellCommandsAndWait(ctx, instruction.PopulateScripts, custom_env, func(bytes []byte) (int, error) {
			return logUploader.Write(bytes)
		}, executor.shouldKillProcesses())
		if err != nil || cmd == nil || cmd.ProcessState == nil || !cmd.ProcessState.Success() {
			message := fmt.Sprintf("\nFailed to execute populate script for %s cache!", commandName)
			executor.cacheAttempts.Failed(cacheKey, message)
			logUploader.Write([]byte(message))
			return false
		}
		executor.cacheAttempts.PopulatedIn(cacheKey, time.Since(populateStartTime))
	} else if !cachePopulated {
		logUploader.Write([]byte(fmt.Sprintf("\nCache miss for %s! No script to populate with.", cacheKey)))
	}

	caches = append(
		caches,
		Cache{
			Name:                     commandName,
			Key:                      cacheKey,
			BaseFolder:               baseFolder,
			PartiallyExpandedFolders: partiallyExpandedFolders,
			FileHasher:               fileHasher,
			SkipUpload:               cacheAvailable && !instruction.ReuploadOnChanges,
			CacheAvailable:           cacheAvailable,
		},
	)
	return true
}

func (executor *Executor) generateCacheKey(
	ctx context.Context,
	logUploader *LogUploader,
	commandName string,
	instruction *api.CacheInstruction,
	custom_env *environment.Environment,
) (string, bool) {
	if instruction.FingerprintKey != "" {
		return instruction.FingerprintKey, true
	}

	cacheKeyHash := sha256.New()

	if len(instruction.FingerprintScripts) > 0 {
		cmd, err := ShellCommandsAndWait(ctx, instruction.FingerprintScripts, custom_env, func(bytes []byte) (int, error) {
			cacheKeyHash.Write(bytes)
			return logUploader.Write(bytes)
		}, executor.shouldKillProcesses())
		if err != nil || !cmd.ProcessState.Success() {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to execute fingerprint script for %s cache!", commandName)))
			return "", false
		}
	} else {
		cacheKeyHash.Write([]byte(custom_env.Get("CIRRUS_TASK_NAME")))
		cacheKeyHash.Write([]byte(custom_env.Get("CI_NODE_INDEX")))
	}

	return fmt.Sprintf("%s-%x", commandName, cacheKeyHash.Sum(nil)), true
}

func (executor *Executor) findLatestAvailableCache(ctx context.Context, uploader *LogUploader, commandName string) *api.CacheInfo {
	cacheInfoRequest := api.CacheInfoRequest{
		TaskIdentification: client.CirrusTaskIdentification,
		CacheKeyPrefixes:   []string{commandName + "-"}, // Use a prefix that will match any cache key for this command
	}

	response, err := client.CirrusClient.CacheInfo(ctx, &cacheInfoRequest)

	if err != nil {
		uploader.Write([]byte(fmt.Sprintf("\nFailed to find latest available cache for %s: %v\n", commandName, err)))
		return nil
	}
	if response != nil && response.Info != nil {
		return response.Info
	}

	uploader.Write([]byte(fmt.Sprintf("\nThere is no other cache entry available for %s\n", commandName)))

	return nil
}

func (executor *Executor) expandAndDeduplicateGlobs(folders []string) ([]string, string) {
	var result []string

	for _, folder := range folders {
		if pathLooksLikeGlob(folder) {
			expandedGlob, err := doublestar.Glob(folder)
			if err != nil {
				return nil, fmt.Sprintf("\nCannot expand cache folder glob '%s': %v\n", folder, err)
			}

			result = append(result, expandedGlob...)
		} else {
			result = append(result, folder)
		}
	}

	// Deduplicate paths to improve UX
	result = DeduplicatePaths(result)

	return result, ""
}

func pathLooksLikeGlob(path string) bool {
	return strings.Contains(path, "*")
}

func (executor *Executor) tryToDownloadAndPopulateCache(
	ctx context.Context,
	logUploader *LogUploader,
	commandName string,
	cacheHost string,
	cacheKey string,
	folderToCache string,
) (bool, bool) { // successfully populated, available remotely
	cacheFile, fetchDuration, err := FetchCache(ctx, logUploader, commandName, cacheHost, cacheKey)
	if err != nil {
		logUploader.Write([]byte(fmt.Sprintf("\nFailed to fetch archive for %s cache: %s!", commandName, err)))
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return false, true
		} else {
			return false, false
		}
	}
	if cacheFile == nil {
		return false, false
	}

	cacheFileInfo, statErr := os.Stat(cacheFile.Name())
	if statErr != nil {
		executor.cacheAttempts.Failed(cacheKey, fmt.Sprintf("failed to determine cache file size: %v", statErr))
	}

	_, _ = logUploader.Write([]byte(fmt.Sprintf("\nCache hit for %s!", cacheKey)))
	unarchiveStartTime := time.Now()
	err = unarchiveCache(cacheFile, folderToCache)
	if err != nil {
		logUploader.Write([]byte(fmt.Sprintf("\nFailed to unarchive %s cache because of %s! Retrying...\n", commandName, err)))
		os.RemoveAll(folderToCache)
		cacheFile, fetchDuration, err = FetchCache(ctx, logUploader, commandName, cacheHost, cacheKey)
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to fetch archive for %s cache: %s!", commandName, err)))
			if err, ok := err.(net.Error); ok && err.Timeout() {
				return false, true
			} else {
				return false, false
			}
		}
		if cacheFile == nil {
			return false, true
		}
		err = unarchiveCache(cacheFile, folderToCache)
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed again to unarchive %s cache because of %s!\n", commandName, err)))
			logUploader.Write([]byte(fmt.Sprintf("\nTreating this failure as a cache miss but won't try to re-upload! Cleaning up %s...\n", folderToCache)))
			os.RemoveAll(folderToCache)
			return false, true
		}
	} else {
		unarchiveDuration := time.Since(unarchiveStartTime)
		if unarchiveDuration > 10*time.Second {
			logUploader.Write([]byte(fmt.Sprintf("\nUnarchived %s cache entry in %f seconds!\n", commandName, unarchiveDuration.Seconds())))
		}
	}

	if statErr == nil {
		executor.cacheAttempts.Hit(cacheKey, uint64(cacheFileInfo.Size()), fetchDuration, time.Since(unarchiveStartTime))
	}

	return true, true
}

func unarchiveCache(
	cacheFile *os.File,
	folderToCache string,
) error {
	defer os.Remove(cacheFile.Name())
	EnsureFolderExists(folderToCache)
	return targz.Unarchive(cacheFile.Name(), folderToCache)
}

func FetchCache(
	ctx context.Context,
	logUploader *LogUploader,
	commandName string,
	cacheHost string,
	cacheKey string,
) (*os.File, time.Duration, error) {
	cacheFile, err := os.CreateTemp(os.TempDir(), commandName)
	if err != nil {
		slog.Error("Failed to create a temp cache file", "command", commandName, "err", err)
		logUploader.Write([]byte(fmt.Sprintf("\nCache miss for %s!", commandName)))
		return nil, 0, err
	}
	defer cacheFile.Close()

	downloadStartTime := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s/%s", cacheHost, cacheKey), nil)
	if err != nil {
		slog.Error("Failed to create a cache request", "command", commandName, "err", err)
		return nil, 0, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		slog.Error("HTTP cache request failed", "command", commandName, "err", err)
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("Unexpected HTTP cache response status", "command", commandName, "status", resp.Status)
		return nil, 0, nil
	}

	bufferedFileWriter := bufio.NewWriter(cacheFile)
	bytesDownloaded, err := bufferedFileWriter.ReadFrom(bufio.NewReader(resp.Body))
	if err != nil {
		slog.Error("Failed to finish downloading cache", "command", commandName, "err", err)
		return nil, 0, err
	}
	err = bufferedFileWriter.Flush()
	if err != nil {
		slog.Error("Failed to flush cache", "command", commandName, "err", err)
		return nil, 0, err
	}
	downloadDuration := time.Since(downloadStartTime)
	if bytesDownloaded < 1024 {
		logUploader.Write([]byte(fmt.Sprintf("\nDownloaded %d bytes.", bytesDownloaded)))
	} else if bytesDownloaded < 1024*1024 {
		logUploader.Write([]byte(fmt.Sprintf("\nDownloaded %dKb.", bytesDownloaded/1024)))
	} else {
		logUploader.Write([]byte(fmt.Sprintf("\nDownloaded %dMb in %fs.", bytesDownloaded/1024/1024, downloadDuration.Seconds())))
	}
	return cacheFile, downloadDuration, nil
}

func (executor *Executor) UploadCache(
	ctx context.Context,
	logUploader *LogUploader,
	commandName string,
	cacheHost string,
	instruction *api.UploadCacheInstruction,
) bool {
	var err error

	cache := FindCache(instruction.CacheName)

	if cache == nil {
		logUploader.Write([]byte(fmt.Sprintf("No cache found for %s!", instruction.CacheName)))
		return false // cache record should always exists
	}

	if cache.SkipUpload {
		logUploader.Write([]byte(fmt.Sprintf("Skipping change detection for %s cache!", instruction.CacheName)))
		return true
	}

	foldersToCache, message := executor.expandAndDeduplicateGlobs(cache.PartiallyExpandedFolders)
	if message != "" {
		logUploader.Write([]byte(message))
		return false
	}

	commaSeparatedFolders := strings.Join(foldersToCache, ", ")

	if allDirsEmpty(foldersToCache) {
		logUploader.Write([]byte(fmt.Sprintf("All cache folders (%s) are empty! Skipping uploading ...", commaSeparatedFolders)))
		return true
	}

	fileHasher := hasher.New()
	for _, folder := range foldersToCache {
		if err := fileHasher.AddFolder(cache.BaseFolder, folder); err != nil {
			logUploader.Write([]byte(fmt.Sprintf("Failed to calculate hash of %s! %s", folder, err)))
			logUploader.Write([]byte("Skipping uploading of cache!"))
			return true
		}
	}

	logUploader.Write([]byte(fmt.Sprintf("SHA for cache folders (%s) is '%s'\n", commaSeparatedFolders, fileHasher.SHA())))

	if fileHasher.SHA() == cache.FileHasher.SHA() {
		logUploader.Write([]byte(fmt.Sprintf("Cache %s hasn't changed! Skipping uploading...", cache.Name)))
		return true
	}
	if cache.FileHasher.Len() != 0 {
		logUploader.Write([]byte(fmt.Sprintf("Cache %s has changed!", cache.Name)))
		logUploader.Write([]byte(fmt.Sprintf("\nList of changes for cache folders (%s):", commaSeparatedFolders)))

		for _, diffEntry := range cache.FileHasher.DiffWithNewer(fileHasher) {
			logUploader.Write([]byte(fmt.Sprintf("\n%s: %s", diffEntry.Type.String(), diffEntry.Path)))
		}
	}

	cacheFile, err := os.CreateTemp("", "")
	if err != nil {
		logUploader.Write([]byte(fmt.Sprintf("\nFailed to create temporary cache file: %v", err)))
		return false
	}
	defer os.Remove(cacheFile.Name())

	archiveStartTime := time.Now()
	err = targz.Archive(cache.BaseFolder, foldersToCache, cacheFile.Name())
	if err != nil {
		logUploader.Write([]byte(fmt.Sprintf("\nFailed to tar caches for %s with %s!", commandName, err)))
		return false
	}
	archivingDuration := time.Since(archiveStartTime)
	fi, err := cacheFile.Stat()
	if err != nil {
		logUploader.Write([]byte(fmt.Sprintf("\nFailed to create caches archive for %s with %s!", commandName, err)))
		return false
	}

	bytesToUpload := fi.Size()

	if bytesToUpload < 1024 {
		logUploader.Write([]byte(fmt.Sprintf("\n%s cache size is %d bytes.", instruction.CacheName, bytesToUpload)))
	} else if bytesToUpload < 1024*1024 {
		logUploader.Write([]byte(fmt.Sprintf("\n%s cache size is %dKb.", instruction.CacheName, bytesToUpload/1024)))
	} else {
		logUploader.Write([]byte(fmt.Sprintf("\n%s cache size is %dMb.", instruction.CacheName, bytesToUpload/1024/1024)))
	}

	cacheURL := fmt.Sprintf("http://%s/%s", cacheHost, url.PathEscape(cache.Key))

	if !cache.CacheAvailable {
		// check if some other task has uploaded the cache already
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, cacheURL, nil)
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to create cache check request to URL %s!", cacheURL)))
			return false
		}
		response, _ := httpClient.Do(req)
		if response != nil && response.StatusCode == http.StatusOK {
			createdByTaskId := response.Header.Get(http_cache.CirrusHeaderCreatedBy)
			if createdByTaskId != "" {
				logUploader.Write([]byte(fmt.Sprintf("\nTask '%s' has already uploaded cache entry %s! Skipping upload...", createdByTaskId, cache.Key)))
			} else {
				logUploader.Write([]byte(fmt.Sprintf("\nSome other task has already uploaded cache entry %s! Skipping upload...", cache.Key)))
			}
			return true
		}
	}

	logUploader.Write([]byte(fmt.Sprintf("\nUploading cache %s...", instruction.CacheName)))
	uploadStartTime := time.Now()
	err = UploadCacheFile(ctx, cacheURL, cacheFile)
	if err != nil {
		logUploader.Write([]byte(fmt.Sprintf("\nFailed to upload cache '%s': %s!", commandName, err)))
		logUploader.Write([]byte("\nIgnoring the error..."))
		return true
	}

	executor.cacheAttempts.Miss(cache.Key, uint64(bytesToUpload), archivingDuration, time.Since(uploadStartTime))

	return true
}

func UploadCacheFile(ctx context.Context, cacheURL string, cacheFile *os.File) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cacheURL, cacheFile)
	if err != nil {
		return err
	}
	fileStat, err := cacheFile.Stat()
	if err != nil {
		return err
	}
	req.ContentLength = fileStat.Size()
	req.Header.Set("Content-Type", "application/octet-stream")
	response, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		return fmt.Errorf("bad response status from HTTP cache %d: %s", response.StatusCode, response.Status)
	}
	return nil
}

func FindCache(cacheName string) *Cache {
	for i := 0; i < len(caches); i++ {
		if caches[i].Name == cacheName {
			return &caches[i]
		}
	}
	return nil
}

func DeduplicatePaths(paths []string) (result []string) {
	sort.Strings(paths)

	var previous string

	for _, path := range paths {
		if previous == "" || !strings.Contains(path, previous+string(os.PathSeparator)) {
			result = append(result, path)
			previous = path
		}
	}

	return
}
