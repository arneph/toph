/*
check Channel.bad state unreachable
*/
A[] not Channel0.bad
/*
check Channel.bad state unreachable
*/
A[] not Channel1.bad
/*
check Channel.bad state unreachable
*/
A[] not Channel2.bad
/*
check Channel.bad state unreachable
*/
A[] not Channel3.bad
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and adder_0.receiving_in_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and adder_0.sending_out_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and adder_1.receiving_in_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and adder_1.sending_out_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and adder_2.receiving_in_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and adder_2.sending_out_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.sending_chOne_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_chOut_0)

