package azureblob

import (
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"

	uploadablepkg "github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/azureblob/uploadable"
	"github.com/getsentry/sentry-go"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/puzpuzpuz/xsync/v3"
	"github.com/samber/lo"
)

const APIMountPoint = "/_azureblob/cirrus-runners-cache"

// As documented in "Status and error codes" documentation[1]
//
// [1]: https://learn.microsoft.com/en-us/rest/api/storageservices/status-and-error-codes2
type statusAndError struct {
	XMLName xml.Name `xml:"Error"`
	Message string   `xml:"Message"`
}

type AzureBlob struct {
	mux                     *http.ServeMux
	uploadables             *xsync.MapOf[string, *uploadablepkg.Uploadable]
	httpClient              *http.Client
	withUnexpectedEOFReader bool
}

func New(httpClient *http.Client, opts ...Option) *AzureBlob {
	azureBlobContainer := &AzureBlob{
		mux:         http.NewServeMux(),
		uploadables: xsync.NewMapOf[string, *uploadablepkg.Uploadable](),
		httpClient:  httpClient,
	}

	// Apply opts
	for _, opt := range opts {
		opt(azureBlobContainer)
	}

	azureBlobContainer.mux.HandleFunc("GET /{key...}", azureBlobContainer.getBlobAbstract)
	azureBlobContainer.mux.HandleFunc("HEAD /{key...}", azureBlobContainer.headBlobAbstract)
	azureBlobContainer.mux.HandleFunc("PUT /{key...}", azureBlobContainer.putBlobAbstract)

	return azureBlobContainer
}

func (azureBlob *AzureBlob) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// Provide "x-ms-request-id" header for libraries like
	// go-actions-cache that expect it in the response[1][2]
	//
	// [1]: https://github.com/tonistiigi/go-actions-cache/blob/378c5ed1ddd9f4bd9371b02deeca46c9b6fae2fa/cache_v2.go#L74
	// [2]: https://github.com/moby/buildkit/blob/a23bc16feff9789f207a7b59220ce79c86444a39/vendor/github.com/tonistiigi/go-actions-cache/cache_v2.go#L73
	writer.Header().Set("x-ms-request-id", uuid.NewString())

	azureBlob.mux.ServeHTTP(writer, request)
}

func fail(writer http.ResponseWriter, request *http.Request, status int, msg string, args ...any) {
	// Report failure to the Sentry
	hub := sentry.GetHubFromContext(request.Context())

	hub.WithScope(func(scope *sentry.Scope) {
		scope.AddEventProcessor(func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// Swap the exception type and value to work around
			// https://github.com/getsentry/sentry/issues/17837
			savedType := event.Exception[0].Type
			event.Exception[0].Type = event.Exception[0].Value
			event.Exception[0].Value = savedType

			return event
		})

		argsAsSentryContext := sentry.Context{}

		for _, chunk := range lo.Chunk(args, 2) {
			key := fmt.Sprintf("%v", chunk[0])

			var value string

			if len(chunk) > 1 {
				value = fmt.Sprintf("%v", chunk[1])
			}

			argsAsSentryContext[key] = value
		}

		scope.SetContext("Arguments", argsAsSentryContext)

		hub.CaptureException(errors.New(msg))
	})

	message := craftAndLogMessage(slog.LevelError, msg, args...)

	if writer == nil {
		return
	}

	// Report failure to the caller
	writer.WriteHeader(status)
	render.XML(writer, request, &statusAndError{
		Message: message,
	})
}

func craftAndLogMessage(level slog.Level, msg string, args ...any) string {
	// Format failure message for non-structured consumers
	var stringBuilder strings.Builder
	logger := slog.New(slog.NewTextHandler(&stringBuilder, nil))
	switch level {
	case slog.LevelDebug:
		logger.Debug(msg, args...)
	case slog.LevelInfo:
		logger.Info(msg, args...)
	case slog.LevelWarn:
		logger.Warn(msg, args...)
	case slog.LevelError:
		logger.Error(msg, args...)
	}
	message := stringBuilder.String()

	// Report failure to the logger
	log.Print(message)

	return message
}
