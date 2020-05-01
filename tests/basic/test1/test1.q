/*
check Channel.bad state unreachable
*/
A[] not Channel0.bad
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and testA_0.sending_b_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and testA_0.sending_b_1)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and testB_0.sending_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and testD_0.sending_ch_0)

