package api

import "strconv"

func OldTaskIdentification(taskID string, clientToken string) *TaskIdentification {
	oldTaskID, err := strconv.ParseInt(taskID, 10, 64)
	if err != nil {
		return nil
	}

	return &TaskIdentification{
		TaskId: oldTaskID,
		Secret: clientToken,
	}
}
