/*
check Channel.bad state unreachable
*/
A[] not Channel0.bad
/*
check Channel.bad state unreachable
*/
A[] not Channel1.bad
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and main.select_pass_2_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and x_0.sending_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and x_0.sending_done_0)

