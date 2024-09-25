package api

import "strconv"

func OldTaskIdentification(taskId string, clientToken string) *TaskIdentification {
	oldTaskId, err := strconv.ParseInt(taskId, 10, 64)
	if err != nil {
		return nil
	}

	return &TaskIdentification{
		TaskId: oldTaskId,
		Secret: clientToken,
	}
}
