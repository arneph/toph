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
A[] not (deadlock and f_0.receiving_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and f_0.receiving_ch_1)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and g_0.receiving_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and g_0.sending_ch_0)

