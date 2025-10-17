package azureblob

type Option func(azureBlob *AzureBlob)

func WithUnexpectedEOFReader() Option {
	return func(azureBlob *AzureBlob) {
		azureBlob.withUnexpectedEOFReader = true
	}
}
