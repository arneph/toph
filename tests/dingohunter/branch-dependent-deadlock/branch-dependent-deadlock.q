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
A[] not (deadlock and R_0.receiving_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and R_0.sending_done_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and R_1.receiving_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and R_1.sending_done_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and R_2.receiving_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and R_2.sending_done_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and S_0.sending_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and S_0.sending_done_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and S_1.sending_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and S_1.sending_done_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and S_2.sending_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and S_2.sending_done_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_done_0)

