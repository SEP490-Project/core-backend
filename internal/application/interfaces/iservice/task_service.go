package iservice

type TaskService interface {
	MarkInProgress(taskID string) error
	SubmitDraft(taskID string, draftContent string) error
	RequestRevision(taskID string, comments string) error //only available after submitted
	ApproveDraft(taskID string) error
	ReleaseDraft(taskID string) error
	MarkCompleted(taskID string) error
}
