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
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and fanin_func62_0.sending_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and fanin_func62_0.sending_c_1)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.range_receiving_cid_var134_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and work_0.sending_out_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and work_1.sending_out_0)

