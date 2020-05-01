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
A[] not (deadlock and fibParallel_0.sending_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and fibParallel_1.sending_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_ch1_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_ch2_0)

