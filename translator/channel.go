package translator

import (
	"fmt"
	"math"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

func (t *translator) channelCount() int {
	channelCount := t.completeFCG.TotalSpecialOpCount(ir.MakeChan)
	if channelCount > t.config.MaxChannelCount {
		channelCount = t.config.MaxChannelCount
	}
	return channelCount
}

func (t *translator) addChannels() {
	if t.channelCount() == 0 {
		return
	}

	t.addChannelProcess()
	t.addChannelDeclarations()
	t.addChannelProcessInstances()
}

func (t *translator) addChannelProcess() {
	proc := t.system.AddProcess("Channel")
	t.channelProcess = proc

	// Parameters:
	proc.AddParameter(fmt.Sprintf("int[0, %d] i", t.channelCount()-1))

	// Queries:
	proc.AddQuery(uppaal.MakeQuery(
		"A[] (not out_of_resources) imply (not $.bad)",
		"check Channel.bad state unreachable", "",
		uppaal.ChannelSafety))

	// States:
	// Open
	idle := proc.AddState("idle", uppaal.NoRenaming)
	idle.SetLocationAndResetNameAndCommentLocation(uppaal.Location{272, 306})

	proc.SetInitialState(idle)

	newSender := proc.AddState("new_sender", uppaal.NoRenaming)
	newSender.SetType(uppaal.Committed)
	newSender.SetLocation(uppaal.Location{102, 306})
	newSender.SetNameLocation(uppaal.Location{8, 274})

	confirmingA := proc.AddState("confirming_a", uppaal.NoRenaming)
	confirmingA.SetType(uppaal.Committed)
	confirmingA.SetLocation(uppaal.Location{102, 442})
	confirmingA.SetNameLocation(uppaal.Location{8, 458})

	newReceiver := proc.AddState("new_receiver", uppaal.NoRenaming)
	newReceiver.SetType(uppaal.Committed)
	newReceiver.SetLocation(uppaal.Location{442, 306})
	newReceiver.SetNameLocation(uppaal.Location{442, 274})

	confirmingB := proc.AddState("confirming_b", uppaal.NoRenaming)
	confirmingB.SetType(uppaal.Committed)
	confirmingB.SetLocation(uppaal.Location{442, 442})
	confirmingB.SetNameLocation(uppaal.Location{442, 458})

	// Closing
	closing := proc.AddState("closing", uppaal.NoRenaming)
	closing.SetType(uppaal.Committed)
	closing.SetLocation(uppaal.Location{272, 85})
	closing.SetNameLocation(uppaal.Location{216, 101})

	// Closed
	closed := proc.AddState("closed", uppaal.NoRenaming)
	closed.SetLocationAndResetNameAndCommentLocation(uppaal.Location{272, -34})

	confirmingClosed := proc.AddState("confirming_closed", uppaal.NoRenaming)
	confirmingClosed.SetType(uppaal.Committed)
	confirmingClosed.SetLocationAndResetNameAndCommentLocation(uppaal.Location{442, -34})

	// Bad
	bad := proc.AddState("bad", uppaal.NoRenaming)
	bad.SetLocation(uppaal.Location{102, -102})
	bad.SetNameLocation(uppaal.Location{54, -134})

	// Transitions:
	// Open, Sender
	trans1 := proc.AddTrans(idle, newSender)
	trans1.SetSync("sender_trigger[i]?")
	trans1.SetSyncLocation(uppaal.Location{129, 306})

	trans2 := proc.AddTrans(newSender, idle)
	trans2.SetGuard("chan_counter[i] > \nchan_buffer[i]")
	trans2.AddNail(uppaal.Location{136, 238})
	trans2.AddNail(uppaal.Location{238, 238})
	trans2.SetGuardLocation(uppaal.Location{129, 206})

	trans3 := proc.AddTrans(newSender, confirmingA)
	trans3.SetGuard("chan_counter[i] <= \nchan_buffer[i]")
	trans3.SetSync("sender_confirm[i]!")
	trans3.SetGuardLocation(uppaal.Location{-42, 342})
	trans3.SetSyncLocation(uppaal.Location{-42, 374})

	trans4 := proc.AddTrans(confirmingA, idle)
	trans4.SetGuard("chan_counter[i] > 0")
	trans4.SetGuardLocation(uppaal.Location{107, 358})

	trans5 := proc.AddTrans(confirmingA, idle)
	trans5.SetGuard("chan_counter[i] <= 0")
	trans5.SetSync("receiver_confirm[i]!")
	trans5.AddNail(uppaal.Location{204, 442})
	trans5.SetGuardLocation(uppaal.Location{118, 442})
	trans5.SetSyncLocation(uppaal.Location{118, 458})

	// Open, Receiver
	trans6 := proc.AddTrans(idle, newReceiver)
	trans6.SetSync("receiver_trigger[i]?")
	trans6.SetSyncLocation(uppaal.Location{298, 306})

	trans7 := proc.AddTrans(newReceiver, idle)
	trans7.SetGuard("chan_counter[i] < 0")
	trans7.AddNail(uppaal.Location{408, 238})
	trans7.AddNail(uppaal.Location{306, 238})
	trans7.SetGuardLocation(uppaal.Location{298, 222})

	trans8 := proc.AddTrans(newReceiver, confirmingB)
	trans8.SetGuard("chan_counter[i] >= 0")
	trans8.SetSync("receiver_confirm[i]!")
	trans8.SetGuardLocation(uppaal.Location{446, 358})
	trans8.SetSyncLocation(uppaal.Location{446, 374})

	trans9 := proc.AddTrans(confirmingB, idle)
	trans9.SetGuard("chan_counter[i] < \nchan_buffer[i]")
	trans9.SetGuardLocation(uppaal.Location{306, 342})

	trans10 := proc.AddTrans(confirmingB, idle)
	trans10.SetGuard("chan_counter[i] >= \nchan_buffer[i]")
	trans10.SetSync("sender_confirm[i]!")
	trans10.AddNail(uppaal.Location{340, 442})
	trans10.SetGuardLocation(uppaal.Location{306, 442})
	trans10.SetSyncLocation(uppaal.Location{306, 474})

	// Closing
	trans11 := proc.AddTrans(idle, closing)
	trans11.SetGuard("chan_counter[i] <= chan_buffer[i]")
	trans11.SetSync("close[i]?")
	trans11.AddUpdate("chan_buffer[i] = -1")
	trans11.SetGuardLocation(uppaal.Location{276, 126})
	trans11.SetSyncLocation(uppaal.Location{276, 142})
	trans11.SetUpdateLocation(uppaal.Location{276, 158})

	trans12 := proc.AddTrans(closing, closing)
	trans12.SetGuard("chan_counter[i] < 0")
	trans12.SetSync("receiver_confirm[i]!")
	trans12.AddUpdate("chan_counter[i]++")
	trans12.AddNail(uppaal.Location{340, 51})
	trans12.AddNail(uppaal.Location{340, 119})
	trans12.SetGuardLocation(uppaal.Location{344, 68})
	trans12.SetSyncLocation(uppaal.Location{344, 84})
	trans12.SetUpdateLocation(uppaal.Location{344, 100})

	trans13 := proc.AddTrans(closing, closed)
	trans13.SetGuard("chan_counter[i] >= 0")
	trans13.SetGuardLocation(uppaal.Location{276, -2})
	trans13.SetUpdateLocation(uppaal.Location{276, 14})

	trans14 := proc.AddTrans(idle, bad)
	trans14.SetGuard("chan_counter[i] > \nchan_buffer[i]")
	trans14.SetSync("close[i]?")
	trans14.AddUpdate("chan_buffer[i] = -1")
	trans14.AddNail(uppaal.Location{272, 170})
	trans14.AddNail(uppaal.Location{102, 170})
	trans14.SetGuardLocation(uppaal.Location{106, 10})
	trans14.SetSyncLocation(uppaal.Location{106, 42})
	trans14.SetUpdateLocation(uppaal.Location{106, 58})

	// Closed
	trans15 := proc.AddTrans(closed, confirmingClosed)
	trans15.SetSync("receiver_trigger[i]?")
	trans15.SetSyncLocation(uppaal.Location{298, -34})

	trans16 := proc.AddTrans(confirmingClosed, closed)
	trans16.SetSync("receiver_confirm[i]!")
	trans16.AddUpdate("chan_counter[i] = (chan_counter[i] >= 0) ? chan_counter[i] : 0")
	trans16.AddNail(uppaal.Location{408, -102})
	trans16.AddNail(uppaal.Location{306, -102})
	trans16.SetSyncLocation(uppaal.Location{298, -118})
	trans16.SetUpdateLocation(uppaal.Location{298, -102})

	trans17 := proc.AddTrans(closed, bad)
	trans17.SetSync("sender_trigger[i]?")
	trans17.AddNail(uppaal.Location{136, -34})
	trans17.SetSyncLocation(uppaal.Location{129, -34})

	trans18 := proc.AddTrans(closed, bad)
	trans18.SetSync("close[i]?")
	trans18.AddNail(uppaal.Location{238, -102})
	trans18.SetSyncLocation(uppaal.Location{129, -118})
}

func (t *translator) addChannelDeclarations() {
	t.system.Declarations().AddVariable("chan_count", "int", "0")
	t.system.Declarations().AddArray("chan_counter", t.channelCount(), "int")
	t.system.Declarations().AddArray("chan_buffer", t.channelCount(), "int")
	t.system.Declarations().AddArray("sender_trigger", t.channelCount(), "chan")
	t.system.Declarations().AddArray("sender_confirm", t.channelCount(), "chan")
	t.system.Declarations().AddArray("receiver_trigger", t.channelCount(), "chan")
	t.system.Declarations().AddArray("receiver_confirm", t.channelCount(), "chan")
	t.system.Declarations().AddArray("close", t.channelCount(), "chan")
	t.system.Declarations().AddFunc(fmt.Sprintf(
		`int make_chan(int buffer) {
	int cid;
	if (chan_count == %d) {
		out_of_resources = true;
		return 0;
	}
	cid = chan_count;
	chan_count++;
	chan_counter[cid] = 0;
	chan_buffer[cid] = buffer;
	return cid;
}`, t.channelCount()))
	t.system.Declarations().AddSpace()
}

func (t *translator) addChannelProcessInstances() {
	c := t.channelCount()
	if c > 1 {
		c--
	}
	d := fmt.Sprintf("%d", int(math.Log10(float64(c))+1))
	for i := 0; i < t.channelCount(); i++ {
		instName := fmt.Sprintf("%s%0"+d+"d", t.channelProcess.Name(), i)
		inst := t.system.AddProcessInstance(t.channelProcess.Name(), instName)
		inst.AddParameter(fmt.Sprintf("%d", i))
	}
}
