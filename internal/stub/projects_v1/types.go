// Package projects_v1 provides stub types for the pms/projects service API.
// The projects service is not available as a GitHub dependency.
package projects_v1

type Project struct {
	Id int64 `json:"id,omitempty"`
}

func (p *Project) GetId() int64 {
	if p != nil {
		return p.Id
	}
	return 0
}
