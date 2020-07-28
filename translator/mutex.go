package translator

import (
	"fmt"
	"math"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

func (t *translator) mutexCount() int {
	mutexCount := t.completeFCG.TotalSpecialOpCount(ir.MakeMutex)
	if mutexCount < 1 {
		mutexCount = 1
	} else if mutexCount > t.config.MaxMutexCount {
		mutexCount = t.config.MaxMutexCount
	}
	return mutexCount
}

func (t *translator) addMutexes() {
	if t.mutexCount() == 0 {
		return
	}

	t.addMutexProcess()
	t.addMutexDeclarations()
	t.addMutexProcessInstances()
}

func (t *translator) addMutexProcess() {
	proc := t.system.AddProcess("Mutex")
	t.mutexProcess = proc

	// Parameters:
	proc.AddParameter(fmt.Sprintf("int[0, %d] i", t.mutexCount()-1))

	// Queries:
	proc.AddQuery(uppaal.NewQuery(
		"A[] (not out_of_resources) imply (not $.bad)",
		"check Mutex.bad state unreachable", "",
		uppaal.MutexSafety))

	// Local Declarations:
	proc.Declarations().AddVariable("active_readers", "int", "0")

	// States:
	idle := proc.AddState("idle", uppaal.NoRenaming)
	idle.SetLocation(uppaal.Location{170, 306})
	idle.SetNameLocation(uppaal.Location{187, 298})

	proc.SetInitialState(idle)

	writeLocked := proc.AddState("write_locked", uppaal.NoRenaming)
	writeLocked.SetLocation(uppaal.Location{0, 306})
	writeLocked.SetNameLocation(uppaal.Location{17, 298})

	readLocking := proc.AddState("read_locking", uppaal.NoRenaming)
	readLocking.SetType(uppaal.Committed)
	readLocking.SetLocation(uppaal.Location{340, 238})
	readLocking.SetNameLocation(uppaal.Location{300, 206})

	readLocked := proc.AddState("read_locked", uppaal.NoRenaming)
	readLocked.SetLocation(uppaal.Location{544, 306})
	readLocked.SetNameLocation(uppaal.Location{561, 298})

	readToWriteLocked := proc.AddState("read_locked_to_write_locked", uppaal.NoRenaming)
	readToWriteLocked.SetType(uppaal.Committed)
	readToWriteLocked.SetLocation(uppaal.Location{170, 476})
	readToWriteLocked.SetNameLocation(uppaal.Location{74, 508})

	bad := proc.AddState("bad", uppaal.NoRenaming)
	bad.SetLocation(uppaal.Location{170, 102})
	bad.SetNameLocation(uppaal.Location{160, 70})

	// Transitions:
	// Idle, Write locked:
	trans1 := proc.AddTransition(idle, writeLocked)
	trans1.SetSync("write_lock[i]?")
	trans1.SetSyncLocation(uppaal.Location{38, 222})
	trans1.AddNail(uppaal.Location{136, 238})
	trans1.AddNail(uppaal.Location{34, 238})

	trans2 := proc.AddTransition(writeLocked, idle)
	trans2.SetSync("write_unlock[i]?")
	trans2.SetSyncLocation(uppaal.Location{38, 374})
	trans2.AddNail(uppaal.Location{34, 374})
	trans2.AddNail(uppaal.Location{136, 374})

	// Idle, Read locked:
	trans3 := proc.AddTransition(idle, readLocking)
	trans3.SetSync("read_lock[i]?")
	trans3.SetSyncLocation(uppaal.Location{208, 222})
	trans3.AddUpdate("active_readers++", false)
	trans3.SetUpdateLocation(uppaal.Location{208, 238})
	trans3.AddNail(uppaal.Location{204, 238})

	trans4 := proc.AddTransition(readLocking, readLocking)
	trans4.SetSync("read_lock[i]?")
	trans4.SetSyncLocation(uppaal.Location{310, 289})
	trans4.AddUpdate("active_readers++", false)
	trans4.SetUpdateLocation(uppaal.Location{310, 305})
	trans4.AddNail(uppaal.Location{374, 289})
	trans4.AddNail(uppaal.Location{306, 289})

	trans5 := proc.AddTransition(readLocking, readLocked)
	trans5.SetGuard("mutex_pending_readers[i] == 0", true)
	trans5.SetGuardLocation(uppaal.Location{357, 222})
	trans5.AddNail(uppaal.Location{510, 238})

	trans6 := proc.AddTransition(readLocked, idle)
	trans6.SetGuard("active_readers == 1 && \nmutex_pending_writers[i] == 0", true)
	trans6.SetGuardLocation(uppaal.Location{208, 342})
	trans6.SetSync("read_unlock[i]?")
	trans6.SetSyncLocation(uppaal.Location{208, 374})
	trans6.AddUpdate("active_readers--", false)
	trans6.SetUpdateLocation(uppaal.Location{208, 390})
	trans6.AddNail(uppaal.Location{510, 374})
	trans6.AddNail(uppaal.Location{204, 374})

	// Read locked:
	trans7 := proc.AddTransition(readLocked, readLocked)
	trans7.SetGuard("mutex_pending_writers[i] == 0", true)
	trans7.SetGuardLocation(uppaal.Location{650, 206})
	trans7.SetSync("read_lock[i]?")
	trans7.SetSyncLocation(uppaal.Location{650, 222})
	trans7.AddUpdate("active_readers++", false)
	trans7.SetUpdateLocation(uppaal.Location{650, 238})
	trans7.AddNail(uppaal.Location{612, 204})
	trans7.AddNail(uppaal.Location{646, 204})
	trans7.AddNail(uppaal.Location{646, 272})

	trans8 := proc.AddTransition(readLocked, readLocked)
	trans8.SetGuard("active_readers > 1", false)
	trans8.SetGuardLocation(uppaal.Location{650, 358})
	trans8.SetSync("read_unlock[i]?")
	trans8.SetSyncLocation(uppaal.Location{650, 374})
	trans8.AddUpdate("active_readers--", false)
	trans8.SetUpdateLocation(uppaal.Location{650, 390})
	trans8.AddNail(uppaal.Location{612, 408})
	trans8.AddNail(uppaal.Location{646, 408})
	trans8.AddNail(uppaal.Location{646, 340})

	// Read locked, Write locked:
	trans9 := proc.AddTransition(readLocked, readToWriteLocked)
	trans9.SetGuard("active_readers == 1 && \nmutex_pending_writers[i] > 0", true)
	trans9.SetGuardLocation(uppaal.Location{208, 444})
	trans9.SetSync("read_unlock[i]?")
	trans9.SetSyncLocation(uppaal.Location{208, 476})
	trans9.AddUpdate("active_readers--", false)
	trans9.SetUpdateLocation(uppaal.Location{208, 492})
	trans9.AddNail(uppaal.Location{544, 476})

	trans10 := proc.AddTransition(readToWriteLocked, writeLocked)
	trans10.SetSync("write_lock[i]?")
	trans10.SetSyncLocation(uppaal.Location{38, 476})
	trans10.AddNail(uppaal.Location{0, 476})

	// Bad:
	trans11 := proc.AddTransition(writeLocked, bad)
	trans11.SetSync("read_unlock[i]?")
	trans11.SetSyncLocation(uppaal.Location{24, 102})
	trans11.AddNail(uppaal.Location{0, 102})

	trans12 := proc.AddTransition(idle, bad)
	trans12.SetSync("read_unlock[i]?")
	trans12.SetSyncLocation(uppaal.Location{24, 162})
	trans12.AddNail(uppaal.Location{170, 238})
	trans12.AddNail(uppaal.Location{136, 204})
	trans12.AddNail(uppaal.Location{136, 136})

	trans13 := proc.AddTransition(idle, bad)
	trans13.SetSync("write_unlock[i]?")
	trans13.SetSyncLocation(uppaal.Location{208, 162})
	trans13.AddNail(uppaal.Location{170, 238})
	trans13.AddNail(uppaal.Location{204, 204})
	trans13.AddNail(uppaal.Location{204, 136})

	trans14 := proc.AddTransition(readLocked, bad)
	trans14.SetSync("write_unlock[i]?")
	trans14.SetSyncLocation(uppaal.Location{208, 102})
	trans14.AddNail(uppaal.Location{544, 102})
}

func (t *translator) addMutexDeclarations() {
	t.system.Declarations().AddVariable("mutex_count", "int", "0")
	t.system.Declarations().AddArray("mutex_pending_readers", []int{t.mutexCount()}, "int")
	t.system.Declarations().AddArray("mutex_pending_writers", []int{t.mutexCount()}, "int")
	t.system.Declarations().AddArray("read_lock", []int{t.mutexCount()}, "chan")
	t.system.Declarations().AddArray("read_unlock", []int{t.mutexCount()}, "chan")
	t.system.Declarations().AddArray("write_lock", []int{t.mutexCount()}, "chan")
	t.system.Declarations().AddArray("write_unlock", []int{t.mutexCount()}, "chan")
	t.system.Declarations().AddSpaceBetweenVariables()

	t.system.Declarations().AddFunc(fmt.Sprintf(
		`int make_mutex() {
	int mid;
	if (mutex_count == %d) {
		out_of_resources = true;
		return 0;
	}
	mid = mutex_count;
	mutex_count++;
	mutex_pending_readers[mid] = 0;
	mutex_pending_writers[mid] = 0;
	return mid;
}`, t.mutexCount()))
}

func (t *translator) addMutexProcessInstances() {
	c := t.mutexCount()
	if c > 1 {
		c--
	}
	d := fmt.Sprintf("%d", int(math.Log10(float64(c))+1))
	for i := 0; i < t.mutexCount(); i++ {
		instName := fmt.Sprintf("%s%0"+d+"d", t.mutexProcess.Name(), i)
		inst := t.system.AddProcessInstance(t.mutexProcess, instName)
		inst.AddParameter(fmt.Sprintf("%d", i))
	}
}
