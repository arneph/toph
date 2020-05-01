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
A[] not (deadlock and main_func393_0.select_pass_2_0)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and main_func394_0.select_pass_2_0)

