package containerbackend

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

func unrollStream(reader io.Reader, logChan chan<- string, errChan chan<- error) {
	buildProgressReader := bufio.NewReader(reader)

	for {
		// Docker build progress is line-based
		line, _, err := buildProgressReader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}

			errChan <- err
			return
		}

		// Each line is a JSON object with the actual message wrapped in it
		msg := &struct {
			Stream string
		}{}
		if err := json.Unmarshal(line, &msg); err != nil {
			errChan <- err
			return
		}

		// We're only interested with messages containing the "stream" field, as these are the most helpful
		if msg.Stream == "" {
			continue
		}

		// Cut the unnecessary formatting done by the Docker daemon for some reason
		progressMessage := strings.TrimSpace(msg.Stream)

		// Some messages contain only "\n", so filter these out
		if progressMessage == "" {
			continue
		}

		logChan <- progressMessage
	}
}
