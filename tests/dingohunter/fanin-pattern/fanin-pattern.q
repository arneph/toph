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
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and fanin_func58_0.select_pass_2_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and fanin_func58_0.sending_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and fanin_func58_0.sending_c_1)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and work_0.sending_out_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and work_1.sending_out_0)

