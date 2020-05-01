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
A[] not (deadlock and main_func423_0.sending_chA_0)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and main_func424_0.select_pass_2_0)

