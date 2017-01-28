package hashring

const (
	// IdentityLabelKey is the key used to identify the identity label of a
	// Member
	IdentityLabelKey = "__identity"
)

// Member defines a member of the membership. It can be used by applications to
// apply specific business logic on Members. Examples are:
// - Get the address of a member for RPC calls, both forwarding of internal
//   calls that should target a Member
// - Decissions to include a Member in a query via predicates.
type Member interface {
	// GetAddress returns the external address used by the rpc layer to
	// communicate to the member.
	//
	// Note: It is prefixed with Get for legacy reasons and can be removed after
	// a refactor of the swim.Member to free up the `Address` name.
	GetAddress() string

	// Label reads the label for a given key from the member. It also returns
	// wether or not the label was present on the member
	Label(key string) (value string, has bool)

	// Identity returns the logical identity the member takes within the
	// hashring, this is experimental and might move away from the membership to
	// the Hashring
	Identity() string
}

// MemberChange shows the state before and after the change of a Member
type MemberChange struct {
	// Before is the state of the member before the change, if the
	// member is a new member the before state is nil
	Before Member
	// After is the state of the member after the change, if the
	// member left the after state will be nil
	After Member
}

// ChangeEvent indicates that the membership has changed. The event will contain
// a list of changes that will show both the old and the new state of a member.
// It is not guaranteed that any of the observable state of a member has in fact
// changed, it might only be an interal state change for the underlying
// membership.
type ChangeEvent struct {
	// Changes is a slice of changes that is related to this event
	Changes []MemberChange
}
