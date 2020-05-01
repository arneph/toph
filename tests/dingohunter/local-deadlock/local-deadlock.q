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
A[] not (deadlock and Recv_0.receiving_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and Recv_0.sending_done_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and Recv_1.receiving_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and Recv_1.sending_done_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and Send_0.sending_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_done_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_done_1)

