package store

// Group kind (Dahusim: subscription list vs manual collection).
const (
	GroupKindSubscription = "subscription"
	GroupKindManual       = "manual"
)

func inferGroupKind(subscriptionLink, name string) string {
	if name == BuiltinWLGroupName {
		return GroupKindManual
	}
	if subscriptionLink != "" {
		return GroupKindSubscription
	}
	return GroupKindManual
}
