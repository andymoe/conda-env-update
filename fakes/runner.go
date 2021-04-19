package fakes

import "sync"

type Runner struct {
	ExecuteCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			CondaEnvPath   string
			CondaCachePath string
			WorkingDir     string
		}
		Returns struct {
			Error error
		}
		Stub func(string, string, string) error
	}
	ShouldRunCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			WorkingDir string
			Metadata   map[string]interface {
			}
		}
		Returns struct {
			Bool   bool
			String string
			Error  error
		}
		Stub func(string, map[string]interface {
		}) (bool, string, error)
	}
}

func (f *Runner) Execute(param1 string, param2 string, param3 string) error {
	f.ExecuteCall.Lock()
	defer f.ExecuteCall.Unlock()
	f.ExecuteCall.CallCount++
	f.ExecuteCall.Receives.CondaEnvPath = param1
	f.ExecuteCall.Receives.CondaCachePath = param2
	f.ExecuteCall.Receives.WorkingDir = param3
	if f.ExecuteCall.Stub != nil {
		return f.ExecuteCall.Stub(param1, param2, param3)
	}
	return f.ExecuteCall.Returns.Error
}
func (f *Runner) ShouldRun(param1 string, param2 map[string]interface {
}) (bool, string, error) {
	f.ShouldRunCall.Lock()
	defer f.ShouldRunCall.Unlock()
	f.ShouldRunCall.CallCount++
	f.ShouldRunCall.Receives.WorkingDir = param1
	f.ShouldRunCall.Receives.Metadata = param2
	if f.ShouldRunCall.Stub != nil {
		return f.ShouldRunCall.Stub(param1, param2)
	}
	return f.ShouldRunCall.Returns.Bool, f.ShouldRunCall.Returns.String, f.ShouldRunCall.Returns.Error
}
