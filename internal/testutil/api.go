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

		// Shun obsolete fields
		task.DeprecatedInstance = nil

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

	res = append(res, '\n')

	return res
}

func TasksFromJSON(t *testing.T, jsonBytes []byte) (result []*api.Task) {
	var jsonTasks []interface{}

	if err := json.Unmarshal(jsonBytes, &jsonTasks); err != nil {
		t.Fatal(err)
	}

	for _, jsonTask := range jsonTasks {
		jsonTaskBytes, err := json.Marshal(jsonTask)
		if err != nil {
			t.Fatal(err)
		}

		var task api.Task

		if err := protojson.Unmarshal(jsonTaskBytes, &task); err != nil {
			t.Fatal(err)
		}

		result = append(result, &task)
	}

	return
}
