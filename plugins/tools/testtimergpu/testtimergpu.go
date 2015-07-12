package testtimergpu

import (
	"github.com/jmmcatee/cracklord/common"
)

type testTimerGPU struct {
	toolUUID string
}

func Setup() error {
	return nil
}

func (h *testTimerGPU) Name() string {
	return "Timer Test Plugin"
}
func (h *testTimerGPU) Type() string {
	return "Test"
}
func (h *testTimerGPU) Version() string {
	return "GPU"
}
func (h *testTimerGPU) UUID() string {
	return h.toolUUID
}
func (h *testTimerGPU) SetUUID(s string) {
	h.toolUUID = s
}

func (h *testTimerGPU) Parameters() string {
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

func (h *testTimerGPU) Requirements() string {
	return common.RES_GPU
}

func (h *testTimerGPU) NewTask(job common.Job) (common.Tasker, error) {
	return newTestTimerTask(job)
}

func NewTooler() common.Tooler {
	return &testTimerGPU{}
}
