package testutil

import (
	"encoding/json"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"google.golang.org/protobuf/encoding/protojson"
	"testing"
)

func TasksToJSON(t *testing.T, tasks []*api.Task) []byte {
	var unmarshalledTasks []interface{}

	for _, task := range tasks {
		var unmarshalledTask interface{}

		marshalled, err := protojson.MarshalOptions{Indent: "  "}.Marshal(task)
		if err != nil {
			t.Fatal(err)
		}

		if err := json.Unmarshal(marshalled, &unmarshalledTask); err != nil {
			t.Fatal(err)
		}

		unmarshalledTasks = append(unmarshalledTasks, unmarshalledTask)
	}

	res, err := json.MarshalIndent(unmarshalledTasks, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	return res
}
