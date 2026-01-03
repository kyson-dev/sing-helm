package service

type ReloadStage string

const (
	ReloadStageStop  ReloadStage = "stop"
	ReloadStageStart ReloadStage = "start"
)

type ReloadError struct {
	Stage ReloadStage
	Err   error
}

func (e *ReloadError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return "reload failed at " + string(e.Stage) + ": " + e.Err.Error()
}

func (e *ReloadError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
