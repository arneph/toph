package translator

import (
	"fmt"
	"strings"

	"github.com/arneph/toph/ir"
)

func (t *translator) structTypeCount(structType *ir.StructType) int {
	structTypeCount := t.completeFCG.TotalStructAllocations(structType)
	if structTypeCount < 1 {
		structTypeCount = 1
	} else if structTypeCount > t.config.MaxStructCount {
		structTypeCount = t.config.MaxStructCount
	}
	return structTypeCount
}

func (t *translator) addType(irType ir.Type) {
	switch irType := irType.(type) {
	case ir.BasicType:
		return
	case *ir.StructType:
		t.addStructType(irType)
	default:
		panic(fmt.Errorf("unexpected ir.Type: %T", irType))
	}
}

func (t *translator) addStructType(structType *ir.StructType) {
	var b1 strings.Builder
	b1.WriteString("typedef struct{\n")
	for _, irField := range structType.Fields() {
		switch irField.Type() {
		case ir.FuncType:
			fmt.Fprintf(&b1, "\tfid %s;\n", irField.Handle())
		default:
			fmt.Fprintf(&b1, "\tint %s;\n", irField.Handle())
		}
	}
	fmt.Fprintf(&b1, "} %s;",
		structType.VariablePrefix())
	t.system.Declarations().AddType(b1.String())
	t.system.Declarations().AddSpaceBetweenTypes()

	t.system.Declarations().AddVariable(
		fmt.Sprintf("%s_count", structType.VariablePrefix()),
		"int", "0")
	t.system.Declarations().AddArray(
		fmt.Sprintf("%s_array", structType.VariablePrefix()),
		t.structTypeCount(structType),
		structType.VariablePrefix())
	t.system.Declarations().AddSpaceBetweenVariables()

	var b2 strings.Builder
	var b3 strings.Builder
	for _, field := range structType.Fields() {
		fmt.Fprintf(&b2, "\t%s_array[sid].%s = %s;\n",
			structType.VariablePrefix(),
			field.Handle(),
			t.translateValue(field.InitialValue(), field.Type()))
		if irFieldStructType, ok := field.Type().(*ir.StructType); ok && !field.IsPointer() {
			fmt.Fprintf(&b3, "\t%[1]s_array[new_sid].%[2]s = copy_%[3]s(%[1]s_array[old_sid].%[2]s);\n",
				structType.VariablePrefix(),
				field.Handle(),
				irFieldStructType.VariablePrefix())
		} else {
			fmt.Fprintf(&b3, "\t%[1]s_array[new_sid].%[2]s = %[1]s_array[old_sid].%[2]s;\n",
				structType.VariablePrefix(),
				field.Handle())
		}
	}

	t.system.Declarations().AddFunc(
		fmt.Sprintf(`int make_%[1]s() {
	int sid;
	if (%[1]s_count == %[2]d) {
		out_of_resources = true;
		return 0;
	}
	sid = %[1]s_count;
	%[1]s_count++;

%[3]s
	return sid;
}`, structType.VariablePrefix(), t.structTypeCount(structType), b2.String()))

	t.system.Declarations().AddFunc(
		fmt.Sprintf(`int copy_%[1]s(int old_sid) {
	int new_sid;
	if (%[1]s_count == %[2]d) {
		out_of_resources = true;
		return 0;
	}
	new_sid = %[1]s_count;
	%[1]s_count++;

%[3]s
	return new_sid;
}`, structType.VariablePrefix(), t.structTypeCount(structType), b3.String()))
}
