package scheduler

import "fmt"

type ErrConnectSchedulerFailure struct {
	Inner   error
	Message string
}
type ErrCreateTaskFailure struct {
	Inner   error
	Message string
}
type ErrRetrieveTasksFailure struct {
	Inner   error
	Message string
}
type ErrRetrieveTaskFolderFailure struct {
	Inner   error
	Message string
}
type ErrDeleteTaskFailure struct {
	Inner   error
	Message string
}
type ErrDeleteTaskFolderFailure struct {
	Inner   error
	Message string
}

func (e *ErrConnectSchedulerFailure) Error() string {
	return fmt.Sprintf("Inner error: %v; Message: %v", e.Inner, e.Message)
}
func (e *ErrCreateTaskFailure) Error() string {
	return fmt.Sprintf("Inner error: %v; Message: %v", e.Inner, e.Message)
}
func (e *ErrRetrieveTasksFailure) Error() string {
	return fmt.Sprintf("Inner error: %v; Message: %v", e.Inner, e.Message)
}
func (e *ErrRetrieveTaskFolderFailure) Error() string {
	return fmt.Sprintf("Inner error: %v; Message: %v", e.Inner, e.Message)
}
func (e *ErrDeleteTaskFailure) Error() string {
	return fmt.Sprintf("Inner error: %v; Message: %v", e.Inner, e.Message)
}
func (e *ErrDeleteTaskFolderFailure) Error() string {
	return fmt.Sprintf("Inner error: %v; Message: %v", e.Inner, e.Message)
}

func (e *ErrConnectSchedulerFailure) Unwrap() error   { return e.Inner }
func (e *ErrCreateTaskFailure) Unwrap() error         { return e.Inner }
func (e *ErrRetrieveTasksFailure) Unwrap() error      { return e.Inner }
func (e *ErrRetrieveTaskFolderFailure) Unwrap() error { return e.Inner }
func (e *ErrDeleteTaskFailure) Unwrap() error         { return e.Inner }
func (e *ErrDeleteTaskFolderFailure) Unwrap() error   { return e.Inner }
