package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
)

func (t *translator) onceCount() int {
	onceCount := t.completeFCG.TotalTypeAllocations(ir.OnceType)
	if onceCount < 1 {
		onceCount = 1
	} else if onceCount > t.config.MaxOnceCount {
		onceCount = t.config.MaxOnceCount
	}
	return onceCount
}

func (t *translator) addOnces() {
	t.system.Declarations().AddVariable("once_count", "int", "0")
	t.system.Declarations().AddArray("once_values", []int{t.onceCount()}, "int")
	t.system.Declarations().AddSpaceBetweenVariables()

	t.system.Declarations().AddFunc(fmt.Sprintf(
		`int make_once() {
	int oid;
	if (once_count == %d) {
		out_of_resources = true;
		return 0;
	}
	oid = once_count;
	once_count++;
	once_values[oid] = 0;
	return oid;
}`, t.onceCount()))
}
