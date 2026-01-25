package ijob

type JobRegistry interface {
	RestartJob(name string) error
}
