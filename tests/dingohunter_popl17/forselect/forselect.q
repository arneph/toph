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
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_done_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_done_1)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and sel1_0.select_pass_2_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and sel1_0.sending_done_0)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and sel2_0.select_pass_2_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and sel2_0.sending_done_0)

