package logger

import (
	"context"
	"encoding/json"
	"incubator/internal/model"
	"log"
	"time"

	"github.com/segmentio/ksuid"
)

type contextKey string

const TaskIdKey contextKey = "taskId"

func getTaskId(ctx context.Context) string {
	if id, ok := ctx.Value(TaskIdKey).(string); ok {
		return id
	}
	return "unknown-task"
}

func LogError(ctx context.Context, resourceId int, action, message string, err error) {
	taskId := getTaskId(ctx)

	logObj := model.LogError{
		Timestamp:  time.Now().Format(time.RFC3339),
		TaskId:     taskId,
		ResourceId: resourceId,
		Action:     action,
		Status:     "failed",
		Message:    message,
		Error:      err,
	}

	jsonData, err := json.Marshal(logObj)
	if err != nil {
		log.Printf("Failed to convert log to JSON: %v", err)
		return
	}

	log.Println(string(jsonData))

}

func LogSuccess(ctx context.Context, resourceId int, action, message, resource string) {
	taskId := getTaskId(ctx)

	logObj := model.LogSuccess{
		Timestamp:  time.Now().Format(time.RFC3339),
		TaskId:     taskId,
		ResourceId: resourceId,
		Action:     action,
		Status:     "success",
		Message:    message,
		Resource:   resource,
	}

	jsonData, err := json.Marshal(logObj)
	if err != nil {
		log.Printf("Failed to convert log to JSON: %v", err)
		return
	}

	log.Println(string(jsonData))

}

func GenerateTaskId() string {
	return ksuid.New().String()
}
