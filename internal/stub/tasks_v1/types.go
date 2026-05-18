// Package tasks_v1 provides stub types for the pms/tasks service API.
// The tasks service is not available as a GitHub dependency.
package tasks_v1

type Task struct {
	Id int64 `json:"id,omitempty"`
}

func (t *Task) GetId() int64 {
	if t != nil {
		return t.Id
	}
	return 0
}
