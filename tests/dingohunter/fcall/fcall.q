/*
check Channel.bad state unreachable
*/
A[] not Channel0.bad
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and g_0.receiving_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and g_1.receiving_ch_0)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and main.select_pass_2_0)

