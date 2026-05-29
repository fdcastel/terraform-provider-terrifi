package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

// applyPlanTestModel is a synthetic model exercising every framework value
// type the real resources use, plus an untagged non-attr.Value field that the
// helper must ignore.
type applyPlanTestModel struct {
	Str      types.String `tfsdk:"str"`
	Bool     types.Bool   `tfsdk:"bool"`
	Int      types.Int64  `tfsdk:"int"`
	List     types.List   `tfsdk:"list"`
	Set      types.Set    `tfsdk:"set"`
	Obj      types.Object `tfsdk:"obj"`
	Untagged string       // no tfsdk tag, not an attr.Value — must be skipped
}

var applyPlanObjAttrs = map[string]attr.Type{"k": types.StringType}

func planList(v string) types.List {
	return types.ListValueMust(types.StringType, []attr.Value{types.StringValue(v)})
}

func planSet(v string) types.Set {
	return types.SetValueMust(types.StringType, []attr.Value{types.StringValue(v)})
}

func planObj(v string) types.Object {
	return types.ObjectValueMust(applyPlanObjAttrs, map[string]attr.Value{"k": types.StringValue(v)})
}

func TestApplyPlanToState(t *testing.T) {
	t.Run("concrete plan overwrites state", func(t *testing.T) {
		plan := &applyPlanTestModel{
			Str:  types.StringValue("new"),
			Bool: types.BoolValue(true),
			Int:  types.Int64Value(42),
			List: planList("a"),
			Set:  planSet("b"),
			Obj:  planObj("c"),
		}
		state := &applyPlanTestModel{
			Str:  types.StringValue("old"),
			Bool: types.BoolValue(false),
			Int:  types.Int64Value(1),
			List: planList("z"),
			Set:  planSet("z"),
			Obj:  planObj("z"),
		}

		applyPlanToState(plan, state)

		assert.Equal(t, "new", state.Str.ValueString())
		assert.True(t, state.Bool.ValueBool())
		assert.Equal(t, int64(42), state.Int.ValueInt64())
		assert.Equal(t, plan.List, state.List)
		assert.Equal(t, plan.Set, state.Set)
		assert.Equal(t, plan.Obj, state.Obj)
	})

	t.Run("null plan fields leave state untouched", func(t *testing.T) {
		plan := &applyPlanTestModel{
			Str:  types.StringNull(),
			Bool: types.BoolNull(),
			Int:  types.Int64Null(),
			List: types.ListNull(types.StringType),
			Set:  types.SetNull(types.StringType),
			Obj:  types.ObjectNull(applyPlanObjAttrs),
		}
		state := &applyPlanTestModel{
			Str:  types.StringValue("keep"),
			Bool: types.BoolValue(true),
			Int:  types.Int64Value(7),
			List: planList("keep"),
			Set:  planSet("keep"),
			Obj:  planObj("keep"),
		}

		applyPlanToState(plan, state)

		assert.Equal(t, "keep", state.Str.ValueString())
		assert.True(t, state.Bool.ValueBool())
		assert.Equal(t, int64(7), state.Int.ValueInt64())
		assert.Equal(t, "keep", state.List.Elements()[0].(types.String).ValueString())
		assert.Equal(t, "keep", state.Set.Elements()[0].(types.String).ValueString())
	})

	t.Run("unknown plan fields leave state untouched", func(t *testing.T) {
		plan := &applyPlanTestModel{
			Str:  types.StringUnknown(),
			Bool: types.BoolUnknown(),
			Int:  types.Int64Unknown(),
			List: types.ListUnknown(types.StringType),
			Set:  types.SetUnknown(types.StringType),
			Obj:  types.ObjectUnknown(applyPlanObjAttrs),
		}
		state := &applyPlanTestModel{
			Str:  types.StringValue("keep"),
			Bool: types.BoolValue(true),
			Int:  types.Int64Value(7),
			List: planList("keep"),
			Set:  planSet("keep"),
			Obj:  planObj("keep"),
		}

		applyPlanToState(plan, state)

		assert.Equal(t, "keep", state.Str.ValueString())
		assert.True(t, state.Bool.ValueBool())
		assert.Equal(t, int64(7), state.Int.ValueInt64())
	})

	t.Run("mixed: only set fields copied", func(t *testing.T) {
		plan := &applyPlanTestModel{
			Str:  types.StringValue("new"),       // set → copied
			Bool: types.BoolNull(),               // null → kept
			Int:  types.Int64Unknown(),           // unknown → kept
			List: planList("a"),                  // set → copied
			Set:  types.SetNull(types.StringType), // null → kept
			Obj:  types.ObjectUnknown(applyPlanObjAttrs), // unknown → kept
		}
		state := &applyPlanTestModel{
			Str:  types.StringValue("old"),
			Bool: types.BoolValue(true),
			Int:  types.Int64Value(7),
			List: planList("z"),
			Set:  planSet("keep"),
			Obj:  planObj("keep"),
		}

		applyPlanToState(plan, state)

		assert.Equal(t, "new", state.Str.ValueString())           // copied
		assert.True(t, state.Bool.ValueBool())                    // kept
		assert.Equal(t, int64(7), state.Int.ValueInt64())         // kept
		assert.Equal(t, "a", state.List.Elements()[0].(types.String).ValueString()) // copied
		assert.Equal(t, "keep", state.Set.Elements()[0].(types.String).ValueString()) // kept
		assert.Equal(t, "keep", state.Obj.Attributes()["k"].(types.String).ValueString()) // kept
	})

	t.Run("untagged non-attr.Value field is ignored", func(t *testing.T) {
		plan := &applyPlanTestModel{Str: types.StringValue("new"), Untagged: "plan"}
		state := &applyPlanTestModel{Str: types.StringValue("old"), Untagged: "state"}

		applyPlanToState(plan, state)

		assert.Equal(t, "new", state.Str.ValueString())
		assert.Equal(t, "state", state.Untagged, "untagged field must not be touched")
	})
}
