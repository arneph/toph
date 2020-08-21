package translator

import (
	"fmt"
	"strings"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

func (t *translator) isTypeUsed(typ ir.Type) bool {
	if !t.config.OptimizeIR {
		return true
	}
	for _, v := range t.vi.VarsUsingType(typ) {
		if t.isVarUsed(v) {
			return true
		}
	}
	for _, f := range t.vi.FuncsUsingType(typ) {
		if t.completeFCG.CalleeCount(f) > 0 {
			return true
		}
	}
	for _, dep := range t.tg.AllTransitiveDependants(typ) {
		if t.isTypeUsed(dep) {
			return true
		}
	}
	return false
}

func (t *translator) structTypeCount(structType *ir.StructType) int {
	structTypeCount := t.completeFCG.TotalTypeAllocations(structType)
	if structTypeCount < 1 {
		structTypeCount = 1
	} else if structTypeCount > t.config.MaxStructCount {
		structTypeCount = t.config.MaxStructCount
	}
	return structTypeCount
}

func (t *translator) containerTypeCount(containerType *ir.ContainerType) int {
	containerTypeCount := t.completeFCG.TotalTypeAllocations(containerType)
	if containerTypeCount < 1 {
		containerTypeCount = 1
	} else if containerTypeCount > t.config.MaxContainerCount {
		containerTypeCount = t.config.MaxContainerCount
	}
	return containerTypeCount
}

func (t *translator) uppaalReferenceTypeForIrType(irType ir.Type) string {
	switch irType {
	case ir.FuncType:
		return "fid"
	default:
		return "int"
	}
}

func (t *translator) addType(irType ir.Type) {
	switch irType := irType.(type) {
	case ir.BasicType:
		switch irType {
		case ir.IntType:
			return
		case ir.FuncType:
			t.system.Declarations().AddType(`typedef struct {
	int id;
	int par_pid;
} fid;

fid make_fid(int id, int par_pid) {
	fid t = {id, par_pid};
	return t;
}`)
			t.system.Declarations().AddSpaceBetweenTypes()
			return
		case ir.ChanType:
			t.addChannels()
		case ir.MutexType:
			t.addMutexes()
		case ir.WaitGroupType:
			t.addWaitGroups()
		case ir.OnceType:
			t.addOnces()
		default:
			panic(fmt.Errorf("unexpected ir.BasicType: %d", irType))
		}
	case *ir.StructType:
		t.addStructType(irType)
	case *ir.ContainerType:
		switch irType.Kind() {
		case ir.Array:
			t.addArrayType(irType)
		case ir.Slice:
			t.addSliceType(irType)
		case ir.Map:
			t.addMapType(irType)
		default:
			panic("unexpected container kind")
		}
	default:
		panic(fmt.Errorf("unexpected ir.Type: %T", irType))
	}
}

func (t *translator) addStructType(structType *ir.StructType) {
	var typeStringBuilder strings.Builder
	typeStringBuilder.WriteString("typedef struct {\n")
	for _, irField := range structType.Fields() {
		fmt.Fprintf(&typeStringBuilder, "\t%s %s;\n",
			t.uppaalReferenceTypeForIrType(irField.Type()),
			irField.Handle())
	}
	fmt.Fprintf(&typeStringBuilder, "} %s;",
		structType.VariablePrefix())
	t.system.Declarations().AddType(typeStringBuilder.String())
	t.system.Declarations().AddSpaceBetweenTypes()

	t.system.Declarations().AddVariable(
		fmt.Sprintf("%s_count", structType.VariablePrefix()),
		"int", "0")
	t.system.Declarations().AddArray(
		fmt.Sprintf("%s_structs", structType.VariablePrefix()),
		[]int{t.structTypeCount(structType)},
		structType.VariablePrefix())
	t.system.Declarations().AddSpaceBetweenVariables()

	var uninitializeFieldsStmts strings.Builder
	var initializeFieldsStmts strings.Builder
	var copyFieldsStmts strings.Builder
	for _, field := range structType.Fields() {
		uninitializedValue := t.translateValue(field.Type().UninitializedValue())
		initializedValue := uninitializedValue
		if !field.IsPointer() {
			initializedValue = t.translateValue(field.Type().InitializedValue())
		}
		fieldHandle := fmt.Sprintf("%s_structs[sid].%s", structType.VariablePrefix(), field.Handle())
		newFieldHandle := fmt.Sprintf("%s_structs[new_sid].%s", structType.VariablePrefix(), field.Handle())
		oldFieldHandle := fmt.Sprintf("%s_structs[old_sid].%s", structType.VariablePrefix(), field.Handle())
		if field.RequiresDeepCopy() {
			oldFieldHandle = t.translateCopyOfRValue(oldFieldHandle, field.Type())
		}
		fmt.Fprintf(&uninitializeFieldsStmts, "\t\t%s = %s;\n", fieldHandle, uninitializedValue)
		fmt.Fprintf(&initializeFieldsStmts, "\t\t%s = %s;\n", fieldHandle, initializedValue)
		fmt.Fprintf(&copyFieldsStmts, "\t%s = %s;\n", newFieldHandle, oldFieldHandle)
	}

	t.system.Declarations().AddFunc(
		fmt.Sprintf(`int make_%[1]s(bool initialize_fields) {
	int sid;
	if (%[1]s_count >= %[2]d) {
		%[1]s_count++;
		out_of_resources = true;
		return 0;
	}
	sid = %[1]s_count;
	%[1]s_count++;

	if (!initialize_fields) {
%[3]s	} else {
%[4]s	}

	return sid;
}`,
			structType.VariablePrefix(),
			t.structTypeCount(structType),
			uninitializeFieldsStmts.String(),
			initializeFieldsStmts.String()))

	t.system.Declarations().AddFunc(
		fmt.Sprintf(`int copy_%[1]s(int old_sid) {
	int new_sid;
	if (%[1]s_count >= %[2]d) {
		%[1]s_count++;
		out_of_resources = true;
		return 0;
	}
	new_sid = %[1]s_count;
	%[1]s_count++;

%[3]s
	return new_sid;
}`,
			structType.VariablePrefix(),
			t.structTypeCount(structType),
			copyFieldsStmts.String()))

	if t.config.GenerateResourceBoundQueries {
		t.system.AddQuery(uppaal.NewQuery(
			fmt.Sprintf("A[] %s_count < %d", structType.VariablePrefix(), t.structTypeCount(structType)+1),
			fmt.Sprintf("check resource bound never reached through %s creation", structType.String()),
			"",
			uppaal.ResourceBoundUnreached))
	}
}

func (t *translator) addArrayType(containerType *ir.ContainerType) {
	t.system.Declarations().AddVariable(
		fmt.Sprintf("%s_count", containerType.VariablePrefix()),
		"int", "0")
	t.system.Declarations().AddArray(
		fmt.Sprintf("%s_arrays", containerType.VariablePrefix()),
		[]int{t.containerTypeCount(containerType), containerType.Len()},
		t.uppaalReferenceTypeForIrType(containerType.ElementType()))
	t.system.Declarations().AddSpaceBetweenVariables()

	uninitializedValue := t.translateValue(containerType.ElementType().UninitializedValue())
	initializedValue := uninitializedValue
	if !containerType.HoldsPointers() {
		initializedValue = t.translateValue(containerType.ElementType().InitializedValue())
	}
	oldElementHandle := fmt.Sprintf("%s_arrays[old_aid][i]", containerType.VariablePrefix())
	if containerType.RequiresDeepCopies() {
		oldElementHandle = t.translateCopyOfRValue(oldElementHandle, containerType.ElementType())
	}

	t.system.Declarations().AddFunc(
		fmt.Sprintf(`int make_%[1]s(bool initialize_elements) {
	int aid;
	if (%[1]s_count >= %[2]d) {
		%[1]s_count++;
		out_of_resources = true;
		return 0;
	}
	aid = %[1]s_count;
	%[1]s_count++;

	if (!initialize_elements) {
		for (i : int[0, %[3]d]) {
			%[1]s_arrays[aid][i] = %[4]s;
		}
	} else {
		for (i : int[0, %[3]d]) {
			%[1]s_arrays[aid][i] = %[5]s;
		}
	}

	return aid;
}`,
			containerType.VariablePrefix(),
			t.containerTypeCount(containerType),
			containerType.Len()-1,
			uninitializedValue,
			initializedValue))

	t.system.Declarations().AddFunc(
		fmt.Sprintf(`int copy_%[1]s(int old_aid) {
	int new_aid;
	if (%[1]s_count >= %[2]d) {
		%[1]s_count++;
		out_of_resources = true;
		return 0;
	}
	new_aid = %[1]s_count;
	%[1]s_count++;

	for (i : int[0, %[3]d]) {
		%[1]s_arrays[new_aid][i] = %[4]s;
	}

	return new_aid;
}`,
			containerType.VariablePrefix(),
			t.containerTypeCount(containerType),
			containerType.Len()-1,
			oldElementHandle))

	if t.config.GenerateResourceBoundQueries {
		t.system.AddQuery(uppaal.NewQuery(
			fmt.Sprintf("A[] %s_count < %d", containerType.VariablePrefix(), t.containerTypeCount(containerType)+1),
			fmt.Sprintf("check resource bound never reached through %s creation", containerType.String()),
			"",
			uppaal.ResourceBoundUnreached))
	}
}

func (t *translator) addSliceType(containerType *ir.ContainerType) {
	t.system.Declarations().AddVariable(
		fmt.Sprintf("%s_count", containerType.VariablePrefix()),
		"int", "0")
	t.system.Declarations().AddArray(
		fmt.Sprintf("%s_lengths", containerType.VariablePrefix()),
		[]int{t.containerTypeCount(containerType)},
		"int")
	t.system.Declarations().AddArray(
		fmt.Sprintf("%s_slices", containerType.VariablePrefix()),
		[]int{t.containerTypeCount(containerType), t.config.ContainerCapacity},
		t.uppaalReferenceTypeForIrType(containerType.ElementType()))
	t.system.Declarations().AddSpaceBetweenVariables()

	uninitializedValue := t.translateValue(containerType.ElementType().UninitializedValue())
	initializedValue := uninitializedValue
	if !containerType.HoldsPointers() {
		initializedValue = t.translateValue(containerType.ElementType().InitializedValue())
	}
	oldElementHandle := fmt.Sprintf("%s_slices[old_bid][i]", containerType.VariablePrefix())
	if containerType.RequiresDeepCopies() {
		oldElementHandle = t.translateCopyOfRValue(oldElementHandle, containerType.ElementType())
	}
	srcElementHandle := fmt.Sprintf("%s_slices[src_bid][i]", containerType.VariablePrefix())
	if containerType.RequiresDeepCopies() {
		srcElementHandle = t.translateCopyOfRValue(srcElementHandle, containerType.ElementType())
	}

	t.system.Declarations().AddFunc(
		fmt.Sprintf(`int make_%[1]s(int length, bool initialize_elements) {
	int bid, i;
	if (%[1]s_count >= %[2]d) {
		%[1]s_count++;
		out_of_resources = true;
		return 0;
	}
	bid = %[1]s_count;
	%[1]s_count++;

	%[1]s_lengths[bid] = length;
	if (!initialize_elements) {
		for (i = 0; i < length; i++) {
			%[1]s_slices[bid][i] = %[3]s;
		}
	} else {
		for (i = 0; i < length; i++) {
			%[1]s_slices[bid][i] = %[4]s;
		}
	}

	return bid;
}`,
			containerType.VariablePrefix(),
			t.containerTypeCount(containerType),
			uninitializedValue,
			initializedValue))

	t.system.Declarations().AddFunc(
		fmt.Sprintf(`int copy_%[1]s(int old_bid) {
	int new_bid, i;
	if (%[1]s_count >= %[2]d) {
		%[1]s_count++;
		out_of_resources = true;
		return 0;
	}
	new_bid = %[1]s_count;
	%[1]s_count++;

	%[1]s_lengths[new_bid] = %[1]s_lengths[new_bid];
	for (i = 0; i < %[1]s_lengths[new_bid]; i++) {
		%[1]s_slices[new_bid][i] = %[3]s;
	}

	return new_bid;
}`,
			containerType.VariablePrefix(),
			t.containerTypeCount(containerType),
			oldElementHandle))

	t.system.Declarations().AddFunc(
		fmt.Sprintf(`void append_%[1]s(int bid, %[2]s value) {
	int index = %[1]s_lengths[bid];
	if (index >= %[3]d) {
		out_of_resources = true;
		return;
	}
	%[1]s_lengths[bid]++;
	%[1]s_slices[bid][index] = value;
}`,
			containerType.VariablePrefix(),
			t.uppaalReferenceTypeForIrType(containerType.ElementType()),
			t.containerTypeCount(containerType)))

	t.system.Declarations().AddFunc(
		fmt.Sprintf(`void copy_between_%[1]s(int dst_bid, int src_bid) {
	int i;
	if (dst_bid == src_bid) {
		return;
	}
	for (i = 0; i < %[1]s_lengths[dst_bid] && i < %[1]s_lengths[src_bid]; i++) {
		%[1]s_slices[dst_bid][i] = %[4]s;
	}
}`,
			containerType.VariablePrefix(),
			t.uppaalReferenceTypeForIrType(containerType.ElementType()),
			t.containerTypeCount(containerType),
			srcElementHandle))

	if t.config.GenerateResourceBoundQueries {
		t.system.AddQuery(uppaal.NewQuery(
			fmt.Sprintf("A[] %s_count < %d", containerType.VariablePrefix(), t.containerTypeCount(containerType)+1),
			fmt.Sprintf("check resource bound never reached through %s creation", containerType.String()),
			"",
			uppaal.ResourceBoundUnreached))
	}
}

func (t *translator) addMapType(containerType *ir.ContainerType) {
	t.system.Declarations().AddVariable(
		fmt.Sprintf("%s_count", containerType.VariablePrefix()),
		"int", "0")
	t.system.Declarations().AddArray(
		fmt.Sprintf("%s_lengths", containerType.VariablePrefix()),
		[]int{t.containerTypeCount(containerType)},
		"int")
	t.system.Declarations().AddArray(
		fmt.Sprintf("%s_maps", containerType.VariablePrefix()),
		[]int{t.containerTypeCount(containerType), t.config.ContainerCapacity},
		t.uppaalReferenceTypeForIrType(containerType.ElementType()))
	t.system.Declarations().AddSpaceBetweenVariables()

	uninitializedValue := t.translateValue(containerType.ElementType().UninitializedValue())
	initializedValue := uninitializedValue
	if !containerType.HoldsPointers() {
		initializedValue = t.translateValue(containerType.ElementType().InitializedValue())
	}
	writeElementHandle := fmt.Sprintf("%s_maps[mid][index]", containerType.VariablePrefix())
	readElementHandle := fmt.Sprintf("%s_maps[mid][index]", containerType.VariablePrefix())
	if containerType.RequiresDeepCopies() {
		readElementHandle = t.translateCopyOfRValue(readElementHandle, containerType.ElementType())
	}
	currentElementHandle := fmt.Sprintf("%s_maps[mid][index]", containerType.VariablePrefix())
	nextElementHandle := fmt.Sprintf("%s_maps[mid][index + 1]", containerType.VariablePrefix())

	t.system.Declarations().AddFunc(
		fmt.Sprintf(`int make_%[1]s() {
	int mid;
	if (%[1]s_count >= %[2]d) {
		%[1]s_count++;
		out_of_resources = true;
		return 0;
	}
	mid = %[1]s_count;
	%[1]s_count++;

	%[1]s_lengths[mid] = 0;

	return mid;
}`,
			containerType.VariablePrefix(),
			t.containerTypeCount(containerType)))

	t.system.Declarations().AddFunc(
		fmt.Sprintf(`%[2]s read_%[1]s(int mid, int index) {
	if (index == -1) {
		return %[3]s;
	}
	return %[4]s;
}`,
			containerType.VariablePrefix(),
			t.uppaalReferenceTypeForIrType(containerType.ElementType()),
			initializedValue,
			readElementHandle))

	t.system.Declarations().AddFunc(
		fmt.Sprintf(`void write_%[1]s(int mid, int index, %[2]s value) {
	if (index >= %[3]d) {
		out_of_resources = true;
		return;
	} else if (index >= %[1]s_lengths[mid]) {
		%[1]s_lengths[mid] = index + 1;
	}
	%[4]s = value;
}`,
			containerType.VariablePrefix(),
			t.uppaalReferenceTypeForIrType(containerType.ElementType()),
			t.config.ContainerCapacity,
			writeElementHandle))

	t.system.Declarations().AddFunc(
		fmt.Sprintf(`void delete_%[1]s(int mid, int index) {
	if (mid < 0 || index < 0) {
		return;
	}
	%[1]s_lengths[mid]--;
	for (index = index; index < %[1]s_lengths[mid]; index++) {
		%[2]s = %[3]s;
	}
}`,
			containerType.VariablePrefix(),
			currentElementHandle,
			nextElementHandle))

	if t.config.GenerateResourceBoundQueries {
		t.system.AddQuery(uppaal.NewQuery(
			fmt.Sprintf("A[] %s_count < %d", containerType.VariablePrefix(), t.containerTypeCount(containerType)+1),
			fmt.Sprintf("check resource bound never reached through %s creation", containerType.String()),
			"",
			uppaal.ResourceBoundUnreached))
	}
}
