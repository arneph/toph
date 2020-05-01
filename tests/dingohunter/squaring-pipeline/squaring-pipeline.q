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
A[] not (deadlock and gen_func243_0.sending_out_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.range_receiving_cid_var440_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and sq_func244_0.range_receiving_cid_var434_in_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and sq_func244_0.sending_out_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and sq_func244_1.range_receiving_cid_var434_in_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and sq_func244_1.sending_out_0)

