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
A[] not (deadlock and R_0.receiving_in_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and S_0.sending_out_0)

