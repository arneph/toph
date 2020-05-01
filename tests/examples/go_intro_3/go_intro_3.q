/*
check Channel.bad state unreachable
*/
A[] not Channel0.bad
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and f_0.sending_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and f_0.sending_ch_1)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and f_0.sending_ch_2)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_ch_1)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_ch_2)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_ch_3)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_ch_4)

