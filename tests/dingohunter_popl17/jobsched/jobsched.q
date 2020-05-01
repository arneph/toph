/*
check Channel.bad state unreachable
*/
A[] not Channel0.bad
/*
check Channel.bad state unreachable
*/
A[] not Channel1.bad
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and producer_0.sending_q_0)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and worker_0.select_pass_2_0)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and worker_1.select_pass_2_0)

