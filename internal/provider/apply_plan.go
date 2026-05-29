package provider

import (
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
)

// applyPlanToState merges a resource's planned values into its current state:
// for every `tfsdk`-tagged field that is a framework value (attr.Value), if the
// plan's value is concrete (neither null nor unknown), it is copied over the
// state's value; otherwise the state value is left as-is.
//
// This is the "copy-if-set" merge that the UniFi full-object PUT pattern needs:
// the API returns complete objects, so on Update we start from current state
// (which carries API-set fields the user never specified) and overlay only the
// fields the plan actually provides. plan and state must be pointers to the
// same struct type.
//
// NOT every resource can use this. Three resources encode intentionally
// different merge behavior and keep bespoke applyPlanToState methods:
//   - client_device: clears (sets to null) fields the user removed, so the API
//     receives an authoritative empty value rather than a stale one.
//   - firewall_policy: propagates null for optional-only fields (copy-if-known,
//     not copy-if-set) to avoid "provider produced inconsistent result" errors.
//   - wlan: same copy-if-known treatment for `passphrase` so switching a WLAN
//     from wpapsk to open clears the stored secret.
// Those three must not be migrated to this helper without reproducing that
// null-propagation, or they will regress. The five resources that do use it
// (client_group, dns_record, firewall_group, firewall_zone, network) are pure
// copy-if-set.
func applyPlanToState[T any](plan, state *T) {
	pv := reflect.ValueOf(plan).Elem()
	sv := reflect.ValueOf(state).Elem()
	pt := pv.Type()

	for i := 0; i < pv.NumField(); i++ {
		// Only consider schema-mapped fields.
		if _, ok := pt.Field(i).Tag.Lookup("tfsdk"); !ok {
			continue
		}
		// Only framework value types expose IsNull/IsUnknown.
		av, ok := pv.Field(i).Interface().(attr.Value)
		if !ok {
			continue
		}
		if av.IsNull() || av.IsUnknown() {
			continue
		}
		sv.Field(i).Set(pv.Field(i))
	}
}
