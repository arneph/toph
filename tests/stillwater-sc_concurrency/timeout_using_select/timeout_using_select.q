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
A[] not (deadlock and time_after_func492_0.sending_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and time_after_func492_1.sending_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and time_after_func492_2.sending_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and timed_process_func490_0.sending_c_0)

