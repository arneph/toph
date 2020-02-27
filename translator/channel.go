package translator

import (
	"fmt"

	"github.com/arneph/toph/xta"
)

func (t *translator) addChannels() {
	t.addChannelProcess()
	t.addChannelDeclarations()
	t.addChannelProcessInstances()
}

const maxChannelCount = 10

func (t *translator) addChannelDeclarations() {
	t.system.Declarations().AddVariableDeclaration(fmt.Sprintf(
		`int chan_count = 0;
int chan_counter[%[1]d];
int chan_buffer[%[1]d];
chan sender_alpha[%[1]d];
chan sender_omega[%[1]d];
chan receiver_alpha[%[1]d];
chan receiver_omega[%[1]d];
chan close[%[1]d];

int make_chan(int buffer) {
    int cid = chan_count;
    chan_count++;
    chan_counter[cid] = 0;
    chan_buffer[cid] = buffer;
    return cid;
}
`, maxChannelCount))
}

func (t *translator) addChannelProcessInstances() {
	for i := 0; i < maxChannelCount; i++ {
		instName := fmt.Sprintf("%s%d", t.channelProcess.Name(), i)
		inst := t.system.AddProcessInstance(
			t.channelProcess.Name(),
			instName, xta.NoRenaming)
		inst.AddParameter(fmt.Sprintf("%d", i))
	}
}

func (t *translator) addChannelProcess() {
	proc := t.system.AddProcess("Channel", xta.NoRenaming)
	t.channelProcess = proc

	// Parameters:
	proc.AddParameter(fmt.Sprintf("int[0, %d] i",
		maxChannelCount-1))

	// States:
	// Open
	idle := proc.AddState("idle", xta.NoRenaming)
	proc.SetInitialState(idle)

	newSender := proc.AddState("new_sender", xta.NoRenaming)
	newSender.SetStateType(xta.Commited)

	confirmingA := proc.AddState("confirming_a", xta.NoRenaming)
	confirmingA.SetStateType(xta.Commited)

	newReceiver := proc.AddState("new_receiver", xta.NoRenaming)
	newReceiver.SetStateType(xta.Commited)

	confirmingB := proc.AddState("confirming_b", xta.NoRenaming)
	confirmingB.SetStateType(xta.Commited)

	// Closing
	confirmingClosing := proc.AddState("confirming_closing", xta.NoRenaming)
	confirmingClosing.SetStateType(xta.Commited)

	closing := proc.AddState("closing", xta.NoRenaming)
	closing.SetStateType(xta.Commited)

	// Closed
	closed := proc.AddState("closed", xta.NoRenaming)

	confirmingClosed := proc.AddState("confirming_closed", xta.NoRenaming)
	confirmingClosed.SetStateType(xta.Commited)

	// Bad
	bad := proc.AddState("bad", xta.NoRenaming)

	// Transitions:
	// Open, Sender
	trans1 := proc.AddTrans(idle, newSender)
	trans1.SetSync("sender_alpha[i]?")
	trans1.AddUpdate("chan_counter[i]++")

	trans2 := proc.AddTrans(newSender, idle)
	trans2.SetGuard("chan_counter[i] > chan_buffer[i]")

	trans3 := proc.AddTrans(newSender, confirmingA)
	trans3.SetGuard("chan_counter[i] <= chan_buffer[i]")
	trans3.SetSync("sender_omega[i]!")

	trans4 := proc.AddTrans(confirmingA, idle)
	trans4.SetGuard("chan_counter[i] > 0")

	trans5 := proc.AddTrans(confirmingA, idle)
	trans5.SetGuard("chan_counter[i] <= 0")
	trans5.SetSync("receiver_omega[i]!")

	// Open, Receiver
	trans6 := proc.AddTrans(idle, newReceiver)
	trans6.SetSync("receiver_alpha[i]?")
	trans6.AddUpdate("chan_counter[i]--")

	trans7 := proc.AddTrans(newReceiver, idle)
	trans7.SetGuard("chan_counter[i] < 0")

	trans8 := proc.AddTrans(newReceiver, confirmingB)
	trans8.SetGuard("chan_counter[i] >= 0")
	trans8.SetSync("receiver_omega[i]!")

	trans9 := proc.AddTrans(confirmingB, idle)
	trans9.SetGuard("chan_counter[i] < chan_buffer[i]")

	trans10 := proc.AddTrans(confirmingB, idle)
	trans10.SetGuard("chan_counter[i] >= chan_buffer[i]")
	trans10.SetSync("sender_omega[i]!")

	// Closing
	trans11 := proc.AddTrans(idle, confirmingClosing)
	trans11.SetGuard("chan_counter[i] < 0")
	trans11.SetSync("close[i]?")

	trans12 := proc.AddTrans(confirmingClosing, confirmingClosing)
	trans12.SetGuard("chan_counter[i] < 0")
	trans12.SetSync("receiver_omega[i]!")
	trans12.AddUpdate("chan_counter[i]++")

	trans13 := proc.AddTrans(confirmingClosing, closing)
	trans13.SetGuard("chan_counter[i] == 0")

	trans14 := proc.AddTrans(idle, closing)
	trans14.SetGuard("0 <= chan_counter[i] && chan_counter[i] <= chan_buffer[i]")
	trans14.SetSync("close[i]?")

	trans15 := proc.AddTrans(idle, bad)
	trans15.SetGuard("chan_counter[i] > chan_buffer[i]")
	trans15.SetSync("close[i]?")

	trans16 := proc.AddTrans(closing, closed)
	trans16.AddUpdate("chan_counter[i] = 1,\nchan_buffer[i] = -1")

	// Closed
	trans17 := proc.AddTrans(closed, confirmingClosed)
	trans17.SetSync("receiver_alpha[i]?")

	trans18 := proc.AddTrans(confirmingClosed, closed)
	trans18.SetSync("receiver_omega[i]!")

	trans19 := proc.AddTrans(closed, bad)
	trans19.SetSync("sender_alpha[i]?")

	trans20 := proc.AddTrans(closed, bad)
	trans20.SetSync("close[i]?")
}
