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
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and main.select_pass_2_0)
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
A[] not (deadlock and main.receiving_done_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main_func248_0.sending_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main_func248_0.sending_done_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and time_after_func250_0.sending_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and time_after_func250_1.sending_ch_0)

