package domain

type Role string

const (
	RoleOperator    Role = "operator"
	RoleReviewer    Role = "reviewer"
	RolePublisher   Role = "publisher"
	RoleRightsAdmin Role = "rights_admin"
	RoleAuditor     Role = "auditor"
	RoleScheduler   Role = "scheduler"
)

type Actor struct {
	ID          ID     `json:"id"`
	DisplayName string `json:"display_name"`
	Role        Role   `json:"role"`
}

func (a Actor) Can(action string) bool {
	grants := map[Role]map[string]bool{
		RoleOperator:    {"campaign.edit": true, "preview.read": true, "note.write": true},
		RoleReviewer:    {"review.decide": true, "preview.read": true},
		RolePublisher:   {"release.publish": true, "release.rollback": true, "release.emergency": true},
		RoleRightsAdmin: {"asset.invalidate": true, "asset.rights": true},
		RoleAuditor:     {"audit.read": true, "audit.export": true},
		RoleScheduler:   {"schedule.execute": true},
	}
	return grants[a.Role][action]
}
