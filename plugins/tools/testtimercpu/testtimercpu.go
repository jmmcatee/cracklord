package testtimercpu

import (
	"github.com/jmmcatee/cracklord/common"
)

type testTimerCPU struct {
	toolUUID string
}

func Setup() error {
	return nil
}

func (h *testTimerCPU) Name() string {
	return "Timer Test Plugin"
}
func (h *testTimerCPU) Type() string {
	return "Test"
}
func (h *testTimerCPU) Version() string {
	return "CPU"
}
func (h *testTimerCPU) UUID() string {
	return h.toolUUID
}
func (h *testTimerCPU) SetUUID(s string) {
	h.toolUUID = s
}

func (h *testTimerCPU) Parameters() string {
	params := `{
		"form": [
  			"seconds",
			"result"
		],
		"schema": {
			"type": "object",
			"properties": {
				"seconds": {
					"title": "Seconds",
					"type": "string",
					"default": "60"
				},
				"result": {
					"title": "Result",
					"type": "string",
					"enum": [
						"Success",
						"Failure",
						"Panic!"
					],
					"default": "Success"
				}
			},
			"required": [
				"name",
				"result"
			]
		}
	}`
	return params
}

func (h *testTimerCPU) Requirements() string {
	return common.RES_CPU
}

func (h *testTimerCPU) NewTask(job common.Job) (common.Tasker, error) {
	return newTestTimerTask(job)
}

func NewTooler() common.Tooler {
	return &testTimerCPU{}
}
