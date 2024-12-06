package main

// basic error
type ScriptFlowError struct {
	msg string
}

// implement error interface
func (s *ScriptFlowError) Error() string {
	return s.msg
}

// node status is not online error
func NewNodeStatusNotOnlineError() error {
	return &ScriptFlowError{"node status is not online"}
}

// task not active error
func NewTaskNotActiveError() error {
	return &ScriptFlowError{"task is not active"}
}

// failed create log file directory error
func NewFailedCreateLogFileDirectoryError() error {
	return &ScriptFlowError{"failed to create log file directory"}
}

// invalid log file name
func NewInvalidLogFileNameError() error {
	return &ScriptFlowError{"invalid log file name"}
}

// failed to parse date from log file name
func NewFailedParseDateFromLogFileNameError() error {
	return &ScriptFlowError{"failed to parse date from log file name"}
}
