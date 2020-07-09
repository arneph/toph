package translator

import (
	"fmt"
	"math"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

func (t *translator) waitGroupCount() int {
	waitGroupCount := t.completeFCG.TotalSpecialOpCount(ir.MakeWaitGroup)
	if waitGroupCount > t.config.MaxWaitGroupCount {
		waitGroupCount = t.config.MaxWaitGroupCount
	}
	return waitGroupCount
}

func (t *translator) addWaitGroups() {
	if t.waitGroupCount() == 0 {
		return
	}

	t.addWaitGroupProcess()
	t.addWaitGroupDeclarations()
	t.addWaitGroupProcessInstances()
}

func (t *translator) addWaitGroupProcess() {
	proc := t.system.AddProcess("WaitGroup")
	t.waitGroupProcess = proc

	// Parameters:
	proc.AddParameter(fmt.Sprintf("int[0, %d] i", t.waitGroupCount()-1))

	// Queries:
	proc.AddQuery(uppaal.MakeQuery(
		"A[] (not out_of_resources) imply (not $.bad)",
		"check WaitGroup.bad state unreachable", "",
		uppaal.WaitGroupSafety))

	// States:
	idle := proc.AddState("idle", uppaal.NoRenaming)
	idle.SetLocation(uppaal.Location{0, 0})
	idle.SetNameLocation(uppaal.Location{17, -8})

	proc.SetInitialState(idle)

	adding := proc.AddState("adding", uppaal.NoRenaming)
	adding.SetType(uppaal.Committed)
	adding.SetLocation(uppaal.Location{238, 0})
	adding.SetNameLocation(uppaal.Location{255, -8})

	active := proc.AddState("active_tasks", uppaal.NoRenaming)
	active.SetLocation(uppaal.Location{442, 0})
	active.SetNameLocation(uppaal.Location{459, -8})

	bad := proc.AddState("bad", uppaal.NoRenaming)
	bad.SetLocation(uppaal.Location{238, -136})
	bad.SetNameLocation(uppaal.Location{255, -144})

	// Transitions:
	// Idle:
	trans1 := proc.AddTrans(idle, idle)
	trans1.SetGuard("wait_group_waiters[i] > 0")
	trans1.SetGuardLocation(uppaal.Location{-248, -16})
	trans1.SetSync("wait[i]!")
	trans1.SetSyncLocation(uppaal.Location{-120, 0})
	trans1.AddNail(uppaal.Location{-68, 34})
	trans1.AddNail(uppaal.Location{-68, -34})

	// Idle, Adding:
	trans2 := proc.AddTrans(idle, adding)
	trans2.SetGuard("wait_group_waiters[i] == 0")
	trans2.SetGuardLocation(uppaal.Location{38, -84})
	trans2.SetSync("add[i]?")
	trans2.SetSyncLocation(uppaal.Location{38, -68})
	trans2.AddNail(uppaal.Location{34, -68})
	trans2.AddNail(uppaal.Location{204, -68})

	trans3 := proc.AddTrans(adding, idle)
	trans3.SetGuard("wait_group_counter[i] == 0")
	trans3.SetGuardLocation(uppaal.Location{38, 52})
	trans3.AddNail(uppaal.Location{204, 68})
	trans3.AddNail(uppaal.Location{34, 68})

	// Adding, Active:
	trans4 := proc.AddTrans(adding, active)
	trans4.SetGuard("wait_group_counter[i] > 0")
	trans4.SetGuardLocation(uppaal.Location{276, -84})
	trans4.AddNail(uppaal.Location{272, -68})
	trans4.AddNail(uppaal.Location{408, -68})

	trans5 := proc.AddTrans(active, adding)
	trans5.SetSync("add[i]?")
	trans5.SetSyncLocation(uppaal.Location{276, 68})
	trans5.AddNail(uppaal.Location{408, 68})
	trans5.AddNail(uppaal.Location{272, 68})

	// Bad:
	trans6 := proc.AddTrans(idle, bad)
	trans6.SetGuard("wait_group_waiters[i] > 0")
	trans6.SetGuardLocation(uppaal.Location{38, -152})
	trans6.SetSync("add[i]?")
	trans6.SetSyncLocation(uppaal.Location{38, -136})
	trans6.AddNail(uppaal.Location{0, -136})

	trans7 := proc.AddTrans(adding, bad)
	trans7.SetGuard("wait_group_counter[i] < 0")
	trans7.SetGuardLocation(uppaal.Location{242, -110})
}

func (t *translator) addWaitGroupDeclarations() {
	t.system.Declarations().AddVariable("wait_group_count", "int", "0")
	t.system.Declarations().AddArray("wait_group_counter", t.waitGroupCount(), "int")
	t.system.Declarations().AddArray("wait_group_waiters", t.waitGroupCount(), "int")
	t.system.Declarations().AddArray("add", t.waitGroupCount(), "chan")
	t.system.Declarations().AddArray("wait", t.waitGroupCount(), "chan")
	t.system.Declarations().AddFunc(fmt.Sprintf(
		`int make_wait_group() {
	int wid;
	if (wait_group_count == %d) {
		out_of_resources = true;
		return 0;
	}
	wid = wait_group_count;
	wait_group_count++;
	wait_group_counter[wid] = 0;
	wait_group_waiters[wid] = 0;
	return wid;
}`, t.waitGroupCount()))
	t.system.Declarations().AddSpace()
}

func (t *translator) addWaitGroupProcessInstances() {
	c := t.waitGroupCount()
	if c > 1 {
		c--
	}
	d := fmt.Sprintf("%d", int(math.Log10(float64(c))+1))
	for i := 0; i < t.waitGroupCount(); i++ {
		instName := fmt.Sprintf("%s%0"+d+"d", t.waitGroupProcess.Name(), i)
		inst := t.system.AddProcessInstance(t.waitGroupProcess.Name(), instName)
		inst.AddParameter(fmt.Sprintf("%d", i))
	}
}
