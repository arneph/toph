/*
check Channel.bad state unreachable
*/
A[] not Channel0.bad
/*
check Channel.bad state unreachable
*/
A[] not Channel1.bad
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and main_func370_0.select_pass_2_0)

