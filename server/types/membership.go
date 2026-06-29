package types

type MembershipRole string

const (
	RoleGroupAdmin MembershipRole = "group_admin"
	RoleMember     MembershipRole = "member"
)

type MembershipStatus string

const (
	MembershipRequested MembershipStatus = "requested"
	MembershipApproved  MembershipStatus = "approved"
	MembershipRejected  MembershipStatus = "rejected"
)

type Membership struct {
	Role   MembershipRole
	Status MembershipStatus
}

func (m Membership) Active() bool { return m.Status == MembershipApproved }

func (m Membership) Satisfies(need MembershipRole) bool {
	if need == RoleGroupAdmin {
		return m.Role == RoleGroupAdmin
	}
	return true
}
